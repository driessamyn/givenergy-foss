package api

import (
	"encoding/json"
	"testing"
	"time"
)

func TestUnmarshalCommunicationDeviceResponse(t *testing.T) {
	raw := `{"data":[{"serial_number":"WG123","type":"GPRS","commission_date":"2023-05-18T00:00:00Z","inverter":{"serial":"ED456","status":"NORMAL","last_online":"2026-04-01T12:00:00Z","commission_date":"2023-05-18T00:00:00Z","info":{"battery_type":"LITHIUM","battery":{"nominal_capacity":186,"nominal_voltage":51.2,"depth_of_discharge":1},"model":"GIV-HY5.0","max_charge_rate":3600,"max_discharge_rate":3600},"warranty":{"type":"Standard Legacy","expiry_date":"2028-05-18T10:34:53Z"},"firmware_version":{"ARM":922,"DSP":922},"connections":{"batteries":[{"module_number":1,"serial":"DZ789","firmware_version":3022,"capacity":{"full":191.79,"design":186},"cell_count":16,"has_usb":false,"nominal_voltage":51.2}],"meters":[{"address":1,"serial_number":22967792,"manufacturer_code":"1984233505","type_code":33,"hardware_version":256,"software_version":517,"baud_rate":9600}]},"flags":["full-power-discharge-in-eco-mode","is-controllable"]},"site_id":32883}],"links":{"first":"https://example.com?page=1","last":"https://example.com?page=1","prev":null,"next":null},"meta":{"current_page":1,"from":1,"last_page":1,"path":"https://example.com","per_page":15,"to":1,"total":1}}`

	var resp CommunicationDeviceResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 device, got %d", len(resp.Data))
	}
	dev := resp.Data[0]
	if dev.SerialNumber != "WG123" {
		t.Errorf("serial_number: got %q, want %q", dev.SerialNumber, "WG123")
	}
	if dev.Type != "GPRS" {
		t.Errorf("type: got %q, want %q", dev.Type, "GPRS")
	}
	if dev.SiteID != 32883 {
		t.Errorf("site_id: got %d, want 32883", dev.SiteID)
	}

	inv := dev.Inverter
	if inv.Serial != "ED456" {
		t.Errorf("inverter serial: got %q, want %q", inv.Serial, "ED456")
	}
	if inv.Info.Model != "GIV-HY5.0" {
		t.Errorf("model: got %q, want %q", inv.Info.Model, "GIV-HY5.0")
	}
	if inv.Info.MaxChargeRate != 3600 {
		t.Errorf("max_charge_rate: got %d, want 3600", inv.Info.MaxChargeRate)
	}
	if inv.FirmwareVersion.ARM != 922 {
		t.Errorf("ARM firmware: got %d, want 922", inv.FirmwareVersion.ARM)
	}
	if len(inv.Connections.Batteries) != 1 {
		t.Errorf("batteries: got %d, want 1", len(inv.Connections.Batteries))
	}
	if len(inv.Flags) != 2 {
		t.Errorf("flags: got %d, want 2", len(inv.Flags))
	}

	if resp.Meta.LastPage != 1 {
		t.Errorf("meta.last_page: got %d, want 1", resp.Meta.LastPage)
	}
	if resp.Meta.Total != 1 {
		t.Errorf("meta.total: got %d, want 1", resp.Meta.Total)
	}
}

func TestUnmarshalDataPointResponse(t *testing.T) {
	raw := `{"data":[{"time":"2026-04-01T23:02:00Z","status":"NORMAL","power":{"solar":{"power":0,"arrays":[{"array":1,"voltage":16,"current":0,"power":0},{"array":2,"voltage":0,"current":0,"power":0}]},"grid":{"voltage":242.1,"current":1.2,"power":3,"frequency":50},"battery":{"percent":45,"power":291,"temperature":17},"consumption":{"power":241},"inverter":{"temperature":30.6,"power":245,"output_voltage":240.4,"output_frequency":49.99,"eps_power":0}},"today":{"solar":0,"grid":{"import":0,"export":0},"battery":{"charge":0,"discharge":0},"consumption":0,"ac_charge":0},"total":{"solar":12317.8,"grid":{"import":8331.7,"export":8026.6},"battery":{"charge":6856.1,"discharge":6856.1},"consumption":11240.8,"ac_charge":5912.7},"is_metered":true}],"links":{"first":null,"last":null,"prev":null,"next":null},"meta":{"current_page":1,"from":1,"last_page":278,"path":"https://example.com","per_page":1,"to":1,"total":278}}`

	var resp DataPointResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 data point, got %d", len(resp.Data))
	}
	dp := resp.Data[0]

	wantTime := time.Date(2026, 4, 1, 23, 2, 0, 0, time.UTC)
	if !dp.Time.Equal(wantTime) {
		t.Errorf("time: got %v, want %v", dp.Time, wantTime)
	}
	if dp.Status != "NORMAL" {
		t.Errorf("status: got %q, want NORMAL", dp.Status)
	}
	if dp.Power.Grid.Voltage != 242.1 {
		t.Errorf("grid voltage: got %v, want 242.1", dp.Power.Grid.Voltage)
	}
	if dp.Power.Battery.Percent != 45 {
		t.Errorf("battery percent: got %v, want 45", dp.Power.Battery.Percent)
	}
	if len(dp.Power.Solar.Arrays) != 2 {
		t.Errorf("solar arrays: got %d, want 2", len(dp.Power.Solar.Arrays))
	}
	if dp.Total.Solar != 12317.8 {
		t.Errorf("total solar: got %v, want 12317.8", dp.Total.Solar)
	}
	if !dp.IsMetered {
		t.Errorf("is_metered: got false, want true")
	}

	if resp.Meta.LastPage != 278 {
		t.Errorf("meta.last_page: got %d, want 278", resp.Meta.LastPage)
	}
	if resp.Meta.Total != 278 {
		t.Errorf("meta.total: got %d, want 278", resp.Meta.Total)
	}
}

func TestUnmarshalEventResponse(t *testing.T) {
	raw := `{"data":[{"event":"BMS Over Voltage","start_time":"2026-02-01T03:27:35Z","end_time":"2026-02-01T03:32:37Z"}],"links":{"first":null,"last":null,"prev":null,"next":null},"meta":{"current_page":1,"from":0,"last_page":96,"per_page":3,"to":0,"total":286}}`

	var resp EventResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 event, got %d", len(resp.Data))
	}
	ev := resp.Data[0]

	if ev.Event != "BMS Over Voltage" {
		t.Errorf("event: got %q, want %q", ev.Event, "BMS Over Voltage")
	}
	wantStart := time.Date(2026, 2, 1, 3, 27, 35, 0, time.UTC)
	if !ev.StartTime.Equal(wantStart) {
		t.Errorf("start_time: got %v, want %v", ev.StartTime, wantStart)
	}
	wantEnd := time.Date(2026, 2, 1, 3, 32, 37, 0, time.UTC)
	if !ev.EndTime.Equal(wantEnd) {
		t.Errorf("end_time: got %v, want %v", ev.EndTime, wantEnd)
	}

	if resp.Meta.LastPage != 96 {
		t.Errorf("meta.last_page: got %d, want 96", resp.Meta.LastPage)
	}
	if resp.Meta.Total != 286 {
		t.Errorf("meta.total: got %d, want 286", resp.Meta.Total)
	}
}
