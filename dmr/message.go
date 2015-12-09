// Package dmr contains generic message structures for the Digital Mobile Radio standard
package dmr

import (
	"fmt"

	"github.com/pd0mz/go-dmr/bit"
)

const (
	InfoPartBits    = 98
	InfoBits        = InfoPartBits * 2
	SlotPartBits    = 10
	SlotBits        = SlotPartBits * 2
	PayloadPartBits = InfoPartBits + SlotPartBits
	PayloadBits     = PayloadPartBits * 2
	SignalBits      = 48
	BurstBits       = PayloadBits + SignalBits
)

// Table 9.2: SYNC Patterns
var SYNCPatterns = map[string]struct {
	ControlMode string
	PDU         string
}{
	"\x07\x05\x05\x0f\x0d\x07\x0d\x0f\x07\x05\x0f\x07": {"BS sourced", "voice"},
	"\x0d\x0f\x0f\x05\x07\x0d\x07\x05\x0d\x0f\x05\x0d": {"BS sourced", "data"},
	"\x07\x0f\x07\x0d\x05\x0d\x0d\x05\x07\x0d\x0f\x0d": {"MS sourced", "voice"},
	"\x0d\x05\x0d\x07\x0f\x07\x07\x0f\x0d\x07\x05\x07": {"MS sourced", "data"},
	"\x07\x07\x0d\x05\x05\x0f\x07\x0d\x0f\x0d\x07\x07": {"MS sourced", "rc sync"},
	"\x05\x0d\x05\x07\x07\x0f\x07\x07\x05\x07\x0f\x0f": {"TDMA direct mode time slot 1", "voice"},
	"\x0f\x07\x0f\x0d\x0d\x05\x0d\x0d\x0f\x0d\x05\x05": {"TDMA direct mode time slot 1", "data"},
	"\x07\x0d\x0f\x0f\x0d\x05\x0f\x05\x05\x0d\x05\x0f": {"TDMA direct mode time slot 2", "voice"},
	"\x0d\x07\x05\x05\x07\x0f\x05\x0f\x0f\x07\x0f\x05": {"TDMA direct mode time slot 2", "data"},
	"\x0d\x0d\x07\x0f\x0f\x05\x0d\x07\x05\x07\x0d\x0d": {"Reserved SYNC pattern", "reserved"},
}

// Burst contains data from a single burst, see 4.2.2 Burst and frame structure
type Burst struct {
	bits bit.Bits
}

func NewBurst(raw []byte) (*Burst, error) {
	if len(raw)*8 != BurstBits {
		return nil, fmt.Errorf("dmr: expected %d bits, got %d", BurstBits, len(raw)*8)
	}
	b := &Burst{}
	b.bits = bit.NewBits(raw)
	return b, nil
}

// Info returns the 196 bits of info in the burst. The data is usually BPTC(196, 96) encoded.
func (b *Burst) Info() bit.Bits {
	// The info is contained in bits 0..98 and 166..216 for a total of 196 bits
	var n = make(bit.Bits, InfoBits)
	copy(n[0:InfoPartBits], b.bits[0:InfoPartBits])
	copy(n[InfoPartBits:InfoBits], b.bits[InfoPartBits+SignalBits+SlotBits:BurstBits])
	return n
}

// Payload returns the 216 bits of payload in the burst.
func (b *Burst) Payload() bit.Bits {
	// The payload is contained in bits 0..108 and 156..264 for a total of 216 bits
	var p = make(bit.Bits, PayloadBits)
	copy(p[0:PayloadPartBits], b.bits[0:PayloadPartBits])
	copy(p[PayloadPartBits:PayloadBits], b.bits[PayloadPartBits+SignalBits:BurstBits])
	return p
}

// Signal returns the 48 bits of signal or SYNC information in the burst.
func (b *Burst) Signal() bit.Bits {
	// The signal bits are contained in bits 108..156 for a total of 48 bits
	var s = make(bit.Bits, SignalBits)
	copy(s, b.bits[PayloadPartBits:PayloadPartBits+SignalBits])
	return s
}

func (b *Burst) SlotType() uint32 {
	/* The slottype is 20 bits, starting after the payload info */
	var s uint32
	for i := InfoPartBits; i < InfoPartBits+SlotBits; i++ {
		var shift = uint32(20 - (i - InfoPartBits))
		s = s | uint32(b.bits[i]<<shift)
	}
	return s
}
