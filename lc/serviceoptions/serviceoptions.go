package serviceoptions

import (
	"fmt"
	"strings"

	dmr "github.com/pd0mz/go-dmr"
)

// Priority Levels
const (
	NoPriority uint8 = iota
	Priority1
	Priority2
	Priority3
)

// PriorityName is a map of priority level to string.
var PriorityName = map[uint8]string{
	NoPriority: "no priority",
	Priority1:  "priority 1",
	Priority2:  "priority 2",
	Priority3:  "priority 3",
}

// ServiceOptions Conforms to ETSI TS 102-361-2 7.2.1
type ServiceOptions struct {
	// Emergency service
	Emergency bool
	// Not defined in document
	Privacy bool
	// Broadcast service (only defined in group calls)
	Broadcast bool
	// Open Voice Call Mode
	OpenVoiceCallMode bool
	// Priority 3 (0b11) is the highest priority
	Priority uint8
}

// Byte packs the service options to a single byte.
func (so *ServiceOptions) Byte() byte {
	var b byte
	if so.Emergency {
		b |= dmr.B00000001
	}
	if so.Privacy {
		b |= dmr.B00000010
	}
	if so.Broadcast {
		b |= dmr.B00010000
	}
	if so.OpenVoiceCallMode {
		b |= dmr.B00100000
	}
	b |= (so.Priority << 6)
	return b
}

// String representatation of the service options.
func (so *ServiceOptions) String() string {
	var part = []string{}
	if so.Emergency {
		part = append(part, "emergency")
	}
	if so.Privacy {
		part = append(part, "privacy")
	}
	if so.Broadcast {
		part = append(part, "broadcast")
	}
	if so.OpenVoiceCallMode {
		part = append(part, "Open Voice Call Mode")
	}
	part = append(part, fmt.Sprintf("%s (%d)", PriorityName[so.Priority], so.Priority))
	return strings.Join(part, ", ")
}

// ParseServiceOptions parses the service options byte.
func ParseServiceOptions(data byte) ServiceOptions {
	return ServiceOptions{
		Emergency:         (data & dmr.B00000001) > 0,
		Privacy:           (data & dmr.B00000010) > 0,
		Broadcast:         (data & dmr.B00010000) > 0,
		OpenVoiceCallMode: (data & dmr.B00100000) > 0,
		Priority:          (data & dmr.B11000000) >> 6,
	}
}
