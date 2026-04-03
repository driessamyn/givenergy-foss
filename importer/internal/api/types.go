package api

import "time"

// Pagination types

// Links represents hypermedia links in paginated responses.
type Links struct {
	First *string `json:"first"`
	Last  *string `json:"last"`
	Prev  *string `json:"prev"`
	Next  *string `json:"next"`
}

// Meta represents pagination metadata.
type Meta struct {
	CurrentPage int `json:"current_page"`
	From        int `json:"from"`
	LastPage    int `json:"last_page"`
	PerPage     int `json:"per_page"`
	To          int `json:"to"`
	Total       int `json:"total"`
}

// CommunicationDevice types

// BatteryInfo holds battery specification data.
type BatteryInfo struct {
	NominalCapacity  float64 `json:"nominal_capacity"`
	NominalVoltage   float64 `json:"nominal_voltage"`
	DepthOfDischarge float64 `json:"depth_of_discharge"`
}

// InverterInfo holds inverter specification data.
type InverterInfo struct {
	BatteryType      string      `json:"battery_type"`
	Battery          BatteryInfo `json:"battery"`
	Model            string      `json:"model"`
	MaxChargeRate    int         `json:"max_charge_rate"`
	MaxDischargeRate int         `json:"max_discharge_rate"`
}

// Warranty holds warranty information.
type Warranty struct {
	Type       string    `json:"type"`
	ExpiryDate time.Time `json:"expiry_date"`
}

// FirmwareVersion holds ARM and DSP firmware versions.
type FirmwareVersion struct {
	ARM int `json:"ARM"`
	DSP int `json:"DSP"`
}

// BatteryModule represents a connected battery module.
type BatteryModule struct {
	ModuleNumber    int     `json:"module_number"`
	Serial          string  `json:"serial"`
	FirmwareVersion int     `json:"firmware_version"`
	Capacity        struct {
		Full   float64 `json:"full"`
		Design float64 `json:"design"`
	} `json:"capacity"`
	CellCount      int     `json:"cell_count"`
	HasUSB         bool    `json:"has_usb"`
	NominalVoltage float64 `json:"nominal_voltage"`
}

// Meter represents a connected meter.
type Meter struct {
	Address          int    `json:"address"`
	SerialNumber     int64  `json:"serial_number"`
	ManufacturerCode string `json:"manufacturer_code"`
	TypeCode         int    `json:"type_code"`
	HardwareVersion  int    `json:"hardware_version"`
	SoftwareVersion  int    `json:"software_version"`
	BaudRate         int    `json:"baud_rate"`
}

// Connections holds battery and meter connections.
type Connections struct {
	Batteries []BatteryModule `json:"batteries"`
	Meters    []Meter         `json:"meters"`
}

// Inverter represents an inverter associated with a communication device.
type Inverter struct {
	Serial          string          `json:"serial"`
	Status          string          `json:"status"`
	LastOnline      time.Time       `json:"last_online"`
	CommissionDate  time.Time       `json:"commission_date"`
	Info            InverterInfo    `json:"info"`
	Warranty        Warranty        `json:"warranty"`
	FirmwareVersion FirmwareVersion `json:"firmware_version"`
	Connections     Connections     `json:"connections"`
	Flags           []string        `json:"flags"`
}

// CommunicationDevice represents a communication device from the GivEnergy API.
type CommunicationDevice struct {
	SerialNumber   string    `json:"serial_number"`
	Type           string    `json:"type"`
	CommissionDate time.Time `json:"commission_date"`
	Inverter       Inverter  `json:"inverter"`
	SiteID         int       `json:"site_id"`
}

// CommunicationDeviceResponse is the paginated response for communication devices.
type CommunicationDeviceResponse struct {
	Data  []CommunicationDevice `json:"data"`
	Links Links                 `json:"links"`
	Meta  Meta                  `json:"meta"`
}

// DataPoint types

// SolarArray represents a single solar array's readings.
type SolarArray struct {
	Array   int     `json:"array"`
	Voltage float64 `json:"voltage"`
	Current float64 `json:"current"`
	Power   float64 `json:"power"`
}

// SolarPower holds solar generation data.
type SolarPower struct {
	Power  float64      `json:"power"`
	Arrays []SolarArray `json:"arrays"`
}

// GridPower holds grid power data.
type GridPower struct {
	Voltage   float64 `json:"voltage"`
	Current   float64 `json:"current"`
	Power     float64 `json:"power"`
	Frequency float64 `json:"frequency"`
}

// BatteryPower holds battery power data.
type BatteryPower struct {
	Percent     float64 `json:"percent"`
	Power       float64 `json:"power"`
	Temperature float64 `json:"temperature"`
}

// ConsumptionPower holds consumption data.
type ConsumptionPower struct {
	Power float64 `json:"power"`
}

// InverterPower holds inverter power data.
type InverterPower struct {
	Temperature     float64 `json:"temperature"`
	Power           float64 `json:"power"`
	OutputVoltage   float64 `json:"output_voltage"`
	OutputFrequency float64 `json:"output_frequency"`
	EPSPower        float64 `json:"eps_power"`
}

// Power holds all power readings at a point in time.
type Power struct {
	Solar       SolarPower       `json:"solar"`
	Grid        GridPower        `json:"grid"`
	Battery     BatteryPower     `json:"battery"`
	Consumption ConsumptionPower `json:"consumption"`
	Inverter    InverterPower    `json:"inverter"`
}

// GridEnergy holds grid import/export totals.
type GridEnergy struct {
	Import float64 `json:"import"`
	Export float64 `json:"export"`
}

// BatteryEnergy holds battery charge/discharge totals.
type BatteryEnergy struct {
	Charge    float64 `json:"charge"`
	Discharge float64 `json:"discharge"`
}

// EnergyTotals holds energy totals (today or all-time).
type EnergyTotals struct {
	Solar       float64       `json:"solar"`
	Grid        GridEnergy    `json:"grid"`
	Battery     BatteryEnergy `json:"battery"`
	Consumption float64       `json:"consumption"`
	ACCharge    float64       `json:"ac_charge"`
}

// DataPoint represents a single inverter data point.
type DataPoint struct {
	Time      time.Time    `json:"time"`
	Status    string       `json:"status"`
	Power     Power        `json:"power"`
	Today     EnergyTotals `json:"today"`
	Total     EnergyTotals `json:"total"`
	IsMetered bool         `json:"is_metered"`
}

// DataPointResponse is the paginated response for data points.
type DataPointResponse struct {
	Data  []DataPoint `json:"data"`
	Links Links       `json:"links"`
	Meta  Meta        `json:"meta"`
}

// Event types

// Event represents an inverter event.
type Event struct {
	Event     string    `json:"event"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

// EventResponse is the paginated response for events.
type EventResponse struct {
	Data  []Event `json:"data"`
	Links Links   `json:"links"`
	Meta  Meta    `json:"meta"`
}
