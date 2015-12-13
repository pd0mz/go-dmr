package dmr

import "github.com/tehmaze/go-dmr/bit"

func ExtractVoiceBits(payload bit.Bits) bit.Bits {
	var b = make(bit.Bits, VoiceBits)
	copy(b[:VoiceHalfBits], payload[:VoiceHalfBits])
	copy(b[VoiceHalfBits:], payload[VoiceHalfBits+SyncBits:])
	return b
}
