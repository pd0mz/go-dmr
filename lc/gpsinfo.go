package lc

import (
	"fmt"

	dmr "github.com/pd0mz/go-dmr"
)

// Data Format
// ref: ETSI TS 102 361-2 7.2.18
const (
	ErrorLT2m uint8 = iota
	ErrorLT20m
	ErrorLT200m
	ErrorLT2km
	ErrorLT20km
	ErrorLE200km
	ErrorGT200km
	ErrorUnknown
)

// PositionErrorName is a map of position error to string.
var PositionErrorName = map[uint8]string{
	ErrorLT2m:    "< 2m",
	ErrorLT20m:   "< 20m",
	ErrorLT200m:  "< 200m",
	ErrorLT2km:   "< 2km",
	ErrorLT20km:  "< 20km",
	ErrorLE200km: "<= 200km",
	ErrorGT200km: "> 200km",
	ErrorUnknown: "unknown",
}

// GpsInfoPDU Conforms to ETSI TS 102 361-2 7.1.1.3
type GpsInfoPDU struct {
	PositionError uint8
	Longitude     uint32
	Latitude      uint32
}

// ParseGpsInfoPDU parse gps info pdu
func ParseGpsInfoPDU(data []byte) (*GpsInfoPDU, error) {
	if len(data) != 7 {
		return nil, fmt.Errorf("dmr/lc/talkeralias: expected 7 bytes, got %d", len(data))
	}

	return &GpsInfoPDU{
		PositionError: (data[0] & dmr.B00001110) >> 1,
		Longitude:     uint32(data[0]&dmr.B00000001)<<24 | uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3]),
		Latitude:      uint32(data[4])<<16 | uint32(data[5])<<8 | uint32(data[6]),
	}, nil
}

// Bytes returns GpsInfoPDU as bytes
func (g *GpsInfoPDU) Bytes() []byte {
	return []byte{
		uint8((g.PositionError&dmr.B00000111)<<1) | uint8((g.Longitude>>24)&dmr.B00000001),
		uint8(g.Longitude >> 16),
		uint8(g.Longitude >> 8),
		uint8(g.Longitude),
		uint8(g.Latitude >> 16),
		uint8(g.Latitude >> 8),
		uint8(g.Latitude),
	}
}

func (g *GpsInfoPDU) String() string {
	return fmt.Sprintf("GpsInfo: [ error: %s lon: %d lat: %d ]",
		PositionErrorName[g.PositionError], g.Longitude, g.Latitude)
}
