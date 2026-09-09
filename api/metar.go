package api

import (
	"encoding/json"
	"fmt"
	"time"
)

type Visib string

func (v *Visib) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*v = Visib(s)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(data, &n); err == nil {
		*v = Visib(n.String())
		return nil
	}
	return fmt.Errorf("unsupported visib value %s", data)
}

type Metar struct {
	IcaoID      string  `json:"icaoId"`
	Name        string  `json:"name"`
	ObsTime     int64   `json:"obsTime"`
	ReportTime  string  `json:"reportTime"`
	ReceiptTime string  `json:"receiptTime"`
	Temp        float64 `json:"temp"`
	Dewp        float64 `json:"dewp"`
	Wdir        float64 `json:"wdir"`
	Wspd        float64 `json:"wspd"`
	Visib       Visib   `json:"visib"`
	Altim       float64 `json:"altim"`
	RawOb       string  `json:"rawOb"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Elev        float64 `json:"elev"`
}

func (m Metar) Time() time.Time {
	return time.Unix(m.ObsTime, 0).UTC()
}
