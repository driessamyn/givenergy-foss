package export

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/driessamyn/givenergy-foss/importer/internal/api"
)

var dataPointHeaders = []string{
	"time", "status",
	"solar_power",
	"solar_array1_voltage", "solar_array1_current", "solar_array1_power",
	"solar_array2_voltage", "solar_array2_current", "solar_array2_power",
	"grid_voltage", "grid_current", "grid_power", "grid_frequency",
	"battery_percent", "battery_power", "battery_temperature",
	"consumption_power",
	"inverter_temperature", "inverter_power", "inverter_output_voltage", "inverter_output_frequency", "inverter_eps_power",
	"today_solar", "today_grid_import", "today_grid_export", "today_battery_charge", "today_battery_discharge", "today_consumption", "today_ac_charge",
	"total_solar", "total_grid_import", "total_grid_export", "total_battery_charge", "total_battery_discharge", "total_consumption", "total_ac_charge",
	"is_metered",
}

func fmtFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func arrayField(arrays []api.SolarArray, idx int, field func(api.SolarArray) float64) string {
	if idx < len(arrays) {
		return fmtFloat(field(arrays[idx]))
	}
	return "0"
}

// WriteDataPointsCSV writes data points as CSV to w.
func WriteDataPointsCSV(w io.Writer, points []api.DataPoint) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(dataPointHeaders); err != nil {
		return fmt.Errorf("write header: %w", err)
	}
	for _, dp := range points {
		row := []string{
			dp.Time.UTC().Format(time.RFC3339),
			dp.Status,
			fmtFloat(dp.Power.Solar.Power),
			arrayField(dp.Power.Solar.Arrays, 0, func(a api.SolarArray) float64 { return a.Voltage }),
			arrayField(dp.Power.Solar.Arrays, 0, func(a api.SolarArray) float64 { return a.Current }),
			arrayField(dp.Power.Solar.Arrays, 0, func(a api.SolarArray) float64 { return a.Power }),
			arrayField(dp.Power.Solar.Arrays, 1, func(a api.SolarArray) float64 { return a.Voltage }),
			arrayField(dp.Power.Solar.Arrays, 1, func(a api.SolarArray) float64 { return a.Current }),
			arrayField(dp.Power.Solar.Arrays, 1, func(a api.SolarArray) float64 { return a.Power }),
			fmtFloat(dp.Power.Grid.Voltage),
			fmtFloat(dp.Power.Grid.Current),
			fmtFloat(dp.Power.Grid.Power),
			fmtFloat(dp.Power.Grid.Frequency),
			fmtFloat(dp.Power.Battery.Percent),
			fmtFloat(dp.Power.Battery.Power),
			fmtFloat(dp.Power.Battery.Temperature),
			fmtFloat(dp.Power.Consumption.Power),
			fmtFloat(dp.Power.Inverter.Temperature),
			fmtFloat(dp.Power.Inverter.Power),
			fmtFloat(dp.Power.Inverter.OutputVoltage),
			fmtFloat(dp.Power.Inverter.OutputFrequency),
			fmtFloat(dp.Power.Inverter.EPSPower),
			fmtFloat(dp.Today.Solar),
			fmtFloat(dp.Today.Grid.Import),
			fmtFloat(dp.Today.Grid.Export),
			fmtFloat(dp.Today.Battery.Charge),
			fmtFloat(dp.Today.Battery.Discharge),
			fmtFloat(dp.Today.Consumption),
			fmtFloat(dp.Today.ACCharge),
			fmtFloat(dp.Total.Solar),
			fmtFloat(dp.Total.Grid.Import),
			fmtFloat(dp.Total.Grid.Export),
			fmtFloat(dp.Total.Battery.Charge),
			fmtFloat(dp.Total.Battery.Discharge),
			fmtFloat(dp.Total.Consumption),
			fmtFloat(dp.Total.ACCharge),
			strconv.FormatBool(dp.IsMetered),
		}
		if err := cw.Write(row); err != nil {
			return fmt.Errorf("write row: %w", err)
		}
	}
	cw.Flush()
	return cw.Error()
}

var eventHeaders = []string{"event", "start_time", "end_time"}

// WriteEventsCSV writes events as CSV to w.
func WriteEventsCSV(w io.Writer, events []api.Event) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(eventHeaders); err != nil {
		return fmt.Errorf("write header: %w", err)
	}
	for _, ev := range events {
		row := []string{
			ev.Event,
			ev.StartTime.UTC().Format(time.RFC3339),
			ev.EndTime.UTC().Format(time.RFC3339),
		}
		if err := cw.Write(row); err != nil {
			return fmt.Errorf("write row: %w", err)
		}
	}
	cw.Flush()
	return cw.Error()
}
