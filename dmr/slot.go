package dmr

import "github.com/pd0mz/go-dmr/bit"

var SlotTypeName = [16]string{
	"PI Header",        // 0000
	"VOICE Header:",    // 0001
	"TLC:",             // 0010
	"CSBK:",            // 0011
	"MBC Header:",      // 0100
	"MBC:",             // 0101
	"DATA Header:",     // 0110
	"RATE 1/2 DATA:",   // 0111
	"RATE 3/4 DATA:",   // 1000
	"Slot idle",        // 1001
	"Rate 1 DATA",      // 1010
	"Unknown/Bad (11)", // 1011
	"Unknown/Bad (12)", // 1100
	"Unknown/Bad (13)", // 1101
	"Unknown/Bad (14)", // 1110
	"Unknown/Bad (15)", // 1111
}

func ExtractSlotType(payload bit.Bits) []byte {
	bits := ExtractSlotTypeBits(payload)
	return bits.Bytes()
}

func ExtractSlotTypeBits(payload bit.Bits) bit.Bits {
	var b = make(bit.Bits, SlotTypeBits)
	copy(b[:SlotTypeHalfBits], payload[InfoHalfBits:InfoHalfBits+SlotTypeHalfBits])
	var o = InfoHalfBits + SlotTypeHalfBits + SyncBits
	copy(b[SlotTypeHalfBits:], payload[o:o+SlotTypeHalfBits])
	return b
}
