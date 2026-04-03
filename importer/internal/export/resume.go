package export

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Status represents the completeness of a date's export.
type Status int

const (
	Missing Status = iota
	Partial
	Complete
)

// dataPointFilePath returns the path for a date's CSV file.
func dataPointFilePath(outputDir string, date time.Time) string {
	return filepath.Join(outputDir, "data-points", date.Format("2006"), date.Format("01"), date.Format("2006-01-02")+".csv")
}

// countCSVDataRows counts non-header rows in a CSV file.
func countCSVDataRows(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	// skip header
	if _, err := r.Read(); err != nil {
		if err == io.EOF {
			return 0, nil
		}
		return 0, fmt.Errorf("read header: %w", err)
	}

	count := 0
	for {
		_, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("read row: %w", err)
		}
		count++
	}
	return count, nil
}

// DateStatus returns whether a date's export is complete, partial, or missing.
func DateStatus(outputDir string, date time.Time, expectedTotal int) (Status, error) {
	path := dataPointFilePath(outputDir, date)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return Missing, nil
	}

	count, err := countCSVDataRows(path)
	if err != nil {
		return Missing, fmt.Errorf("count rows for %s: %w", date.Format("2006-01-02"), err)
	}

	if count == expectedTotal {
		return Complete, nil
	}
	return Partial, nil
}

// CompletedDates returns the set of dates (YYYY-MM-DD) that have complete CSV files.
// A file is considered complete if its row count matches the number of entries in a full day
// as determined by comparing the file to itself (all rows present, no partial write).
// Since expectedTotal is not available here, we use a heuristic: a file exists and is non-empty.
// Callers who need strict completeness should use DateStatus directly.
func CompletedDates(outputDir string) (map[string]bool, error) {
	base := filepath.Join(outputDir, "data-points")
	completed := make(map[string]bool)

	years, err := filepath.Glob(filepath.Join(base, "*"))
	if err != nil {
		return nil, fmt.Errorf("glob years: %w", err)
	}

	for _, yearDir := range years {
		months, err := filepath.Glob(filepath.Join(yearDir, "*"))
		if err != nil {
			return nil, fmt.Errorf("glob months: %w", err)
		}
		for _, monthDir := range months {
			files, err := filepath.Glob(filepath.Join(monthDir, "*.csv"))
			if err != nil {
				return nil, fmt.Errorf("glob csv files: %w", err)
			}
			for _, f := range files {
				base := filepath.Base(f)
				date := base[:len(base)-4] // strip .csv
				count, err := countCSVDataRows(f)
				if err != nil {
					return nil, fmt.Errorf("count rows for %s: %w", f, err)
				}
				if count > 0 {
					completed[date] = true
				}
			}
		}
	}

	return completed, nil
}
