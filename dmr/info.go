package dmr

import "github.com/pd0mz/go-dmr/bit"

func ExtractInfoBits(payload bit.Bits) bit.Bits {
	var b = make(bit.Bits, InfoBits)
	copy(b[:InfoHalfBits], payload[:InfoHalfBits])
	copy(b[InfoHalfBits:], payload[InfoHalfBits+SignalBits+SlotTypeBits:])
	return b
}
