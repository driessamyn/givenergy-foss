package export

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/driessamyn/givenergy-foss/importer/internal/api"
)

// DateRange represents a from/to date range.
type DateRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// ExportMetadata holds metadata about a completed export run.
type ExportMetadata struct {
	ExportedAt   time.Time        `json:"exported_at"`
	Inverters    []api.InverterInfo `json:"inverters"`
	DateRange    DateRange        `json:"date_range"`
	TotalDays    int              `json:"total_days"`
	TotalPoints  int              `json:"total_points"`
	TotalEvents  int              `json:"total_events"`
	DurationSecs float64          `json:"duration_secs"`
}

// WriteMetadata writes metadata as JSON to path, creating parent directories as needed.
func WriteMetadata(path string, meta ExportMetadata) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create metadata dir: %w", err)
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}
	return nil
}
