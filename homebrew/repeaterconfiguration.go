package homebrew

import (
	"fmt"

	"github.com/tehmaze/go-dmr"
)

// RepeaterConfiguration holds information about the current repeater. It
// should be returned by a callback in the implementation, returning actual
// information about the current repeater status.
type RepeaterConfiguration struct {
	Callsign    string
	ID          uint32 // Our RepeaterID
	RXFreq      uint32
	TXFreq      uint32
	TXPower     uint8
	ColorCode   uint8
	Latitude    float32
	Longitude   float32
	Height      uint16
	Location    string
	Description string
	URL         string
}

// Bytes returns the configuration as bytes.
func (r *RepeaterConfiguration) Bytes() []byte {
	return []byte(r.String())
}

// String returns the configuration as string.
func (r *RepeaterConfiguration) String() string {
	if r.ColorCode < 1 {
		r.ColorCode = 1
	}
	if r.ColorCode > 15 {
		r.ColorCode = 15
	}
	if r.TXPower > 99 {
		r.TXPower = 99
	}

	var lat = fmt.Sprintf("%-08f", r.Latitude)
	if len(lat) > 8 {
		lat = lat[:8]
	}
	var lon = fmt.Sprintf("%-09f", r.Longitude)
	if len(lon) > 9 {
		lon = lon[:9]
	}

	var b = "RPTC"
	b += fmt.Sprintf("%-8s", r.Callsign)
	b += fmt.Sprintf("%08x", r.ID)
	b += fmt.Sprintf("%09d", r.RXFreq)
	b += fmt.Sprintf("%09d", r.TXFreq)
	b += fmt.Sprintf("%02d", r.TXPower)
	b += fmt.Sprintf("%02d", r.ColorCode)
	b += lat
	b += lon
	b += fmt.Sprintf("%03d", r.Height)
	b += fmt.Sprintf("%-20s", r.Location)
	b += fmt.Sprintf("%-20s", r.Description)
	b += fmt.Sprintf("%-124s", r.URL)
	b += fmt.Sprintf("%-40s", dmr.SoftwareID)
	b += fmt.Sprintf("%-40s", dmr.PackageID)
	return b
}

// ConfigFunc returns an actual RepeaterConfiguration instance when called.
// This is used by the DMR repeater to poll for current configuration,
// statistics and metrics.
type ConfigFunc func() *RepeaterConfiguration
