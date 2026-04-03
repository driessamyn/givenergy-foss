package export

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/driessamyn/givenergy-foss/importer/internal/api"
)

func TestWriteDataPointsCSV_Headers(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteDataPointsCSV(&buf, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := csv.NewReader(&buf)
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row (header), got %d", len(rows))
	}
	want := strings.Join(dataPointHeaders, ",")
	got := strings.Join(rows[0], ",")
	if got != want {
		t.Errorf("headers mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

func TestWriteDataPointsCSV_RoundTrip(t *testing.T) {
	ts := time.Date(2026, 4, 1, 23, 2, 0, 0, time.UTC)
	dp := api.DataPoint{
		Time:   ts,
		Status: "NORMAL",
		Power: api.Power{
			Solar: api.SolarPower{
				Power: 100.5,
				Arrays: []api.SolarArray{
					{Array: 1, Voltage: 16.1, Current: 0.5, Power: 8.05},
					{Array: 2, Voltage: 0, Current: 0, Power: 0},
				},
			},
			Grid:        api.GridPower{Voltage: 242.1, Current: 1.2, Power: 3, Frequency: 50},
			Battery:     api.BatteryPower{Percent: 45, Power: 291, Temperature: 17},
			Consumption: api.ConsumptionPower{Power: 241},
			Inverter:    api.InverterPower{Temperature: 30.6, Power: 245, OutputVoltage: 240.4, OutputFrequency: 49.99, EPSPower: 0},
		},
		Today: api.EnergyTotals{
			Solar:       1.5,
			Grid:        api.GridEnergy{Import: 0.2, Export: 0.1},
			Battery:     api.BatteryEnergy{Charge: 0.3, Discharge: 0.4},
			Consumption: 1.2,
			ACCharge:    0,
		},
		Total: api.EnergyTotals{
			Solar:       12317.8,
			Grid:        api.GridEnergy{Import: 8331.7, Export: 8026.6},
			Battery:     api.BatteryEnergy{Charge: 6856.1, Discharge: 6856.1},
			Consumption: 11240.8,
			ACCharge:    5912.7,
		},
		IsMetered: true,
	}

	var buf bytes.Buffer
	if err := WriteDataPointsCSV(&buf, []api.DataPoint{dp}); err != nil {
		t.Fatalf("write: %v", err)
	}

	r := csv.NewReader(&buf)
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows (header+data), got %d", len(rows))
	}
	data := rows[1]

	checks := map[int]string{
		0:  ts.Format(time.RFC3339),
		1:  "NORMAL",
		2:  "100.5",
		3:  "16.1",  // array1 voltage
		4:  "0.5",   // array1 current
		5:  "8.05",  // array1 power
		6:  "0",     // array2 voltage
		7:  "0",     // array2 current
		8:  "0",     // array2 power
		9:  "242.1", // grid voltage
		36: "true",  // is_metered
	}
	for col, want := range checks {
		if data[col] != want {
			t.Errorf("col %d (%s): got %q, want %q", col, dataPointHeaders[col], data[col], want)
		}
	}
}

func TestWriteDataPointsCSV_MissingSolarArrays(t *testing.T) {
	dp := api.DataPoint{
		Time:   time.Now().UTC(),
		Status: "NORMAL",
	}

	var buf bytes.Buffer
	if err := WriteDataPointsCSV(&buf, []api.DataPoint{dp}); err != nil {
		t.Fatalf("write: %v", err)
	}

	r := csv.NewReader(&buf)
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	data := rows[1]
	// array fields should be "0" when arrays are missing
	for _, col := range []int{3, 4, 5, 6, 7, 8} {
		if data[col] != "0" {
			t.Errorf("col %d (%s): expected 0 for missing array, got %q", col, dataPointHeaders[col], data[col])
		}
	}
}

func TestWriteDataPointsCSV_Empty(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteDataPointsCSV(&buf, []api.DataPoint{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := csv.NewReader(&buf)
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("expected only header row, got %d rows", len(rows))
	}
}

func TestWriteEventsCSV(t *testing.T) {
	start := time.Date(2026, 2, 1, 3, 27, 35, 0, time.UTC)
	end := time.Date(2026, 2, 1, 3, 32, 37, 0, time.UTC)
	events := []api.Event{
		{Event: "BMS Over Voltage", StartTime: start, EndTime: end},
	}

	var buf bytes.Buffer
	if err := WriteEventsCSV(&buf, events); err != nil {
		t.Fatalf("write: %v", err)
	}

	r := csv.NewReader(&buf)
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0][0] != "event" || rows[0][1] != "start_time" || rows[0][2] != "end_time" {
		t.Errorf("unexpected header: %v", rows[0])
	}
	if rows[1][0] != "BMS Over Voltage" {
		t.Errorf("event: got %q", rows[1][0])
	}
	if rows[1][1] != start.Format(time.RFC3339) {
		t.Errorf("start_time: got %q, want %q", rows[1][1], start.Format(time.RFC3339))
	}
	if rows[1][2] != end.Format(time.RFC3339) {
		t.Errorf("end_time: got %q, want %q", rows[1][2], end.Format(time.RFC3339))
	}
}

func TestWriteEventsCSV_Empty(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteEventsCSV(&buf, []api.Event{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := csv.NewReader(&buf)
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("expected only header row, got %d rows", len(rows))
	}
}
