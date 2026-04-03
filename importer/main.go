package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/driessamyn/givenergy-foss/importer/internal/api"
	"github.com/driessamyn/givenergy-foss/importer/internal/export"
)

var version = "dev"

const pageSize = 2000

func main() {
	tokenFlag := flag.String("token", "", "GivEnergy API token (or GIVENERGY_API_TOKEN env var)")
	outputDir := flag.String("output", "./data", "Output directory")
	fromStr := flag.String("from", "", "Start date YYYY-MM-DD (default: inverter commission date)")
	toStr := flag.String("to", "", "End date YYYY-MM-DD (default: yesterday)")
	rateLimit := flag.Int("rate-limit", 250, "Max requests per minute")
	dryRun := flag.Bool("dry-run", false, "Show export plan without fetching data")
	verbose := flag.Bool("verbose", false, "Verbose logging")
	versionFlag := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Println(version)
		return
	}

	// Token resolution: flag > env var > error
	tok := *tokenFlag
	if tok == "" {
		tok = os.Getenv("GIVENERGY_API_TOKEN")
	}
	if tok == "" {
		fmt.Fprintln(os.Stderr, "error: API token required — use --token flag or set GIVENERGY_API_TOKEN env var")
		os.Exit(1)
	}

	ctx := context.Background()
	client := api.NewClient(tok, *rateLimit)

	// Discover inverters
	devices, err := client.DiscoverInverters(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: discovering inverters: %v\n", err)
		os.Exit(1)
	}
	if len(devices) == 0 {
		fmt.Fprintln(os.Stderr, "error: no inverters found")
		os.Exit(1)
	}

	// Parse --to date (default: yesterday in UTC)
	toDate := time.Now().UTC().Truncate(24 * time.Hour).AddDate(0, 0, -1)
	if *toStr != "" {
		toDate, err = time.Parse("2006-01-02", *toStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: invalid --to date %q: %v\n", *toStr, err)
			os.Exit(1)
		}
	}

	// Parse optional --from date (applied to all inverters; per-inverter default is commission date)
	var globalFrom *time.Time
	if *fromStr != "" {
		t, err := time.Parse("2006-01-02", *fromStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: invalid --from date %q: %v\n", *fromStr, err)
			os.Exit(1)
		}
		globalFrom = &t
	}

	startTime := time.Now()
	var totalDays, totalPoints, totalEvents int
	var failedDates []string
	var inverterInfos []api.InverterInfo
	var overallFrom time.Time
	anyFailed := false

	for _, dev := range devices {
		inv := dev.Inverter
		inverterInfos = append(inverterInfos, inv.Info)

		// Determine from date for this inverter
		fromDate := inv.CommissionDate.UTC().Truncate(24 * time.Hour)
		if globalFrom != nil {
			fromDate = *globalFrom
		}

		// Track earliest from for metadata
		if overallFrom.IsZero() || fromDate.Before(overallFrom) {
			overallFrom = fromDate
		}

		fmt.Printf("Inverter: %s (%s), commissioned %s\n",
			inv.Serial, inv.Info.Model, inv.CommissionDate.Format("2006-01-02"))

		// Check completed dates for this inverter (scoped by serial)
		inverterOutputDir := filepath.Join(*outputDir, inv.Serial)
		completed, err := export.CompletedDates(inverterOutputDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not read completed dates for %s: %v\n", inv.Serial, err)
			completed = make(map[string]bool)
		}

		// Build list of dates needing export
		var datesToFetch []time.Time
		for d := fromDate; !d.After(toDate); d = d.AddDate(0, 0, 1) {
			if !completed[d.Format("2006-01-02")] {
				datesToFetch = append(datesToFetch, d)
			}
		}
		totalRange := int(toDate.Sub(fromDate).Hours()/24) + 1
		alreadyDone := totalRange - len(datesToFetch)

		if *dryRun {
			// Estimate: 1 request per day (data points fit in one page at pageSize=2000) + 1 for events
			estimatedReqs := len(datesToFetch) + 1
			estimatedSecs := float64(estimatedReqs) / float64(*rateLimit) * 60.0
			fmt.Printf("  Dry-run plan for %s:\n", inv.Serial)
			fmt.Printf("    Date range:         %s to %s\n", fromDate.Format("2006-01-02"), toDate.Format("2006-01-02"))
			fmt.Printf("    Dates to fetch:     %d\n", len(datesToFetch))
			fmt.Printf("    Already complete:   %d\n", alreadyDone)
			fmt.Printf("    Estimated requests: %d\n", estimatedReqs)
			fmt.Printf("    Estimated time:     %s (at %d req/min)\n", formatDuration(time.Duration(estimatedSecs*float64(time.Second))), *rateLimit)
			continue
		}

		if len(datesToFetch) == 0 {
			fmt.Fprintf(os.Stderr, "  All %d dates already exported, skipping %s\n", alreadyDone, inv.Serial)
			continue
		}

		fmt.Fprintf(os.Stderr, "Exporting %s (%s)\n", inv.Serial, inv.Info.Model)

		numDates := len(datesToFetch)
		for i, d := range datesToFetch {
			dateStr := d.Format("2006-01-02")
			printProgress(numDates, i, startTime)

			points, expectedTotal, fetchErr := client.GetDataPoints(ctx, inv.Serial, dateStr, pageSize)
			if fetchErr != nil {
				clearProgress()
				fmt.Fprintf(os.Stderr, "  error %s: %v\n", dateStr, fetchErr)
				failedDates = append(failedDates, fmt.Sprintf("%s: %v", dateStr, fetchErr))
				anyFailed = true
				continue
			}

			// Verify we got all data points (detect partial fetches)
			if len(points) != expectedTotal && expectedTotal > 0 {
				clearProgress()
				fmt.Fprintf(os.Stderr, "  warning %s: got %d of %d points\n", dateStr, len(points), expectedTotal)
			}

			outPath := filepath.Join(inverterOutputDir, "data-points",
				d.Format("2006"), d.Format("01"), dateStr+".csv")
			if mkErr := os.MkdirAll(filepath.Dir(outPath), 0o755); mkErr != nil {
				clearProgress()
				fmt.Fprintf(os.Stderr, "  error creating dir for %s: %v\n", dateStr, mkErr)
				failedDates = append(failedDates, fmt.Sprintf("%s: %v", dateStr, mkErr))
				anyFailed = true
				continue
			}
			f, mkErr := os.Create(outPath)
			if mkErr != nil {
				clearProgress()
				fmt.Fprintf(os.Stderr, "  error creating file for %s: %v\n", dateStr, mkErr)
				failedDates = append(failedDates, fmt.Sprintf("%s: %v", dateStr, mkErr))
				anyFailed = true
				continue
			}
			writeErr := export.WriteDataPointsCSV(f, points)
			f.Close()
			if writeErr != nil {
				clearProgress()
				fmt.Fprintf(os.Stderr, "  error writing CSV for %s: %v\n", dateStr, writeErr)
				failedDates = append(failedDates, fmt.Sprintf("%s: %v", dateStr, writeErr))
				anyFailed = true
				continue
			}

			totalPoints += len(points)
			totalDays++
			completed[dateStr] = true

			if *verbose {
				clearProgress()
				fmt.Fprintf(os.Stderr, "  %s: %d points\n", dateStr, len(points))
			}
		}

		clearProgress()

		// Fetch events
		fmt.Fprintf(os.Stderr, "  Fetching events for %s...\n", inv.Serial)
		events, evErr := client.GetEvents(ctx, inv.Serial, pageSize)
		if evErr != nil {
			fmt.Fprintf(os.Stderr, "  warning: could not fetch events: %v\n", evErr)
		} else {
			eventsPath := filepath.Join(inverterOutputDir, "events", "events.csv")
			if mkErr := os.MkdirAll(filepath.Dir(eventsPath), 0o755); mkErr != nil {
				fmt.Fprintf(os.Stderr, "  warning: could not create events dir: %v\n", mkErr)
			} else if ef, mkErr := os.Create(eventsPath); mkErr != nil {
				fmt.Fprintf(os.Stderr, "  warning: could not create events file: %v\n", mkErr)
			} else {
				writeErr := export.WriteEventsCSV(ef, events)
				ef.Close()
				if writeErr != nil {
					fmt.Fprintf(os.Stderr, "  warning: could not write events CSV: %v\n", writeErr)
				} else {
					totalEvents += len(events)
				}
			}
		}
	}

	if *dryRun {
		return
	}

	// Write metadata
	elapsed := time.Since(startTime)
	metaPath := filepath.Join(*outputDir, "export-metadata.json")
	meta := export.ExportMetadata{
		ExportedAt:   time.Now().UTC(),
		Inverters:    inverterInfos,
		DateRange:    export.DateRange{From: overallFrom, To: toDate},
		TotalDays:    totalDays,
		TotalPoints:  totalPoints,
		TotalEvents:  totalEvents,
		DurationSecs: elapsed.Seconds(),
	}
	if wErr := export.WriteMetadata(metaPath, meta); wErr != nil {
		fmt.Fprintf(os.Stderr, "warning: could not write metadata: %v\n", wErr)
	}

	// Print summary
	fmt.Printf("\nExport complete:\n")
	fmt.Printf("  Days exported: %d\n", totalDays)
	fmt.Printf("  Data points:   %d\n", totalPoints)
	fmt.Printf("  Events:        %d\n", totalEvents)
	fmt.Printf("  Duration:      %s\n", formatDuration(elapsed))

	if len(failedDates) > 0 {
		fmt.Fprintf(os.Stderr, "\nFailed dates (%d):\n", len(failedDates))
		for _, fd := range failedDates {
			fmt.Fprintf(os.Stderr, "  %s\n", fd)
		}
	}

	if anyFailed {
		os.Exit(1)
	}
}

// printProgress writes an in-place progress bar to stderr.
// done is the number of completed items; total is the total count.
func printProgress(total, done int, startTime time.Time) {
	if total == 0 {
		return
	}
	pct := done * 100 / total
	const barWidth = 26
	filled := barWidth * done / total
	var bar strings.Builder
	for i := 0; i < barWidth; i++ {
		switch {
		case i < filled:
			bar.WriteByte('=')
		case i == filled:
			bar.WriteByte('>')
		default:
			bar.WriteByte(' ')
		}
	}
	remaining := ""
	if done > 0 {
		elapsed := time.Since(startTime)
		perItem := elapsed / time.Duration(done)
		left := perItem * time.Duration(total-done)
		remaining = fmt.Sprintf("  ~%s remaining", formatDuration(left))
	}
	fmt.Fprintf(os.Stderr, "\r  [%s]  %d/%d days  (%d%%)%s",
		bar.String(), done, total, pct, remaining)
}

// clearProgress overwrites the progress line with spaces.
func clearProgress() {
	fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", 80))
}

// formatDuration formats a duration as a human-readable string.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}
