package export

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/driessamyn/givenergy-foss/importer/internal/api"
)

func writeDateCSV(t *testing.T, outputDir string, date time.Time, points []api.DataPoint) {
	t.Helper()
	path := dataPointFilePath(outputDir, date)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer f.Close()
	if err := WriteDataPointsCSV(f, points); err != nil {
		t.Fatalf("write csv: %v", err)
	}
}

func makePoints(n int) []api.DataPoint {
	base := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	points := make([]api.DataPoint, n)
	for i := range points {
		points[i] = api.DataPoint{
			Time:   base.Add(time.Duration(i) * 5 * time.Minute),
			Status: "NORMAL",
		}
	}
	return points
}

func TestDateStatus_Missing(t *testing.T) {
	dir := t.TempDir()
	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	status, err := DateStatus(dir, date, 288)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != Missing {
		t.Errorf("expected Missing, got %v", status)
	}
}

func TestDateStatus_Complete(t *testing.T) {
	dir := t.TempDir()
	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	points := makePoints(288)
	writeDateCSV(t, dir, date, points)

	status, err := DateStatus(dir, date, 288)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != Complete {
		t.Errorf("expected Complete, got %v", status)
	}
}

func TestDateStatus_Partial(t *testing.T) {
	dir := t.TempDir()
	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	points := makePoints(100)
	writeDateCSV(t, dir, date, points)

	status, err := DateStatus(dir, date, 288)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != Partial {
		t.Errorf("expected Partial, got %v", status)
	}
}

func TestDateStatus_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	// Write file with only header (0 data rows)
	writeDateCSV(t, dir, date, []api.DataPoint{})

	status, err := DateStatus(dir, date, 288)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != Partial {
		t.Errorf("expected Partial for 0 rows vs 288 expected, got %v", status)
	}
}

func TestCompletedDates_Empty(t *testing.T) {
	dir := t.TempDir()
	result, err := CompletedDates(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}

func TestCompletedDates_MultipleFiles(t *testing.T) {
	dir := t.TempDir()

	dates := []time.Time{
		time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC),
	}

	for _, d := range dates {
		writeDateCSV(t, dir, d, makePoints(5))
	}

	// Write an empty file for another date — should not appear as complete
	emptyDate := time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)
	writeDateCSV(t, dir, emptyDate, []api.DataPoint{})

	result, err := CompletedDates(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, d := range dates {
		key := d.Format("2006-01-02")
		if !result[key] {
			t.Errorf("expected %s to be in completed dates", key)
		}
	}

	emptyKey := emptyDate.Format("2006-01-02")
	if result[emptyKey] {
		t.Errorf("expected %s (empty file) not to be in completed dates", emptyKey)
	}
}

func TestDataPointFilePath(t *testing.T) {
	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	got := dataPointFilePath("/output", date)
	want := "/output/data-points/2026/04/2026-04-01.csv"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestWriteDataPointsCSV_ColumnCount(t *testing.T) {
	dp := api.DataPoint{Time: time.Now().UTC(), Status: "NORMAL"}
	var buf bytes.Buffer
	if err := WriteDataPointsCSV(&buf, []api.DataPoint{dp}); err != nil {
		t.Fatalf("write: %v", err)
	}
	lines := bytes.Split(bytes.TrimRight(buf.Bytes(), "\n"), []byte("\n"))
	for i, line := range lines {
		cols := bytes.Split(line, []byte(","))
		if len(cols) != len(dataPointHeaders) {
			t.Errorf("line %d: got %d cols, want %d", i, len(cols), len(dataPointHeaders))
		}
	}
}

func TestCompletedDates_PathFormat(t *testing.T) {
	dir := t.TempDir()
	date := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	writeDateCSV(t, dir, date, makePoints(1))

	result, err := CompletedDates(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the key format is YYYY-MM-DD
	key := fmt.Sprintf("%04d-%02d-%02d", date.Year(), int(date.Month()), date.Day())
	if !result[key] {
		t.Errorf("expected key %q in results, got: %v", key, result)
	}
}
