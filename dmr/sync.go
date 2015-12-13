package dmr

import (
	"bytes"

	"github.com/tehmaze/go-dmr/bit"
)

// Table 9.2: SYNC Patterns
const (
	SyncPatternBSSourcedVoice uint16 = 1 << iota
	SyncPatternBSSourcedData
	SyncPatternMSSourcedVoice
	SyncPatternMSSourcedData
	SyncPatternMSSourcedRC
	SyncPatternDirectVoiceTS1
	SyncPatternDirectDataTS1
	SyncPatternDirectVoiceTS2
	SyncPatternDirectDataTS2
	SyncPatternUnknown
)

var (
	bsSourcedVoice  = []byte{0x75, 0x5f, 0xd7, 0xdf, 0x75, 0xf7}
	bsSourcedData   = []byte{0xdf, 0xf5, 0x7d, 0x75, 0xdf, 0x5d}
	msSourcedVoice  = []byte{0x7f, 0x7d, 0x5d, 0xd5, 0x7d, 0xfd}
	msSourcedData   = []byte{0xd5, 0xd7, 0xf7, 0x7f, 0xd7, 0x57}
	msSourcedRC     = []byte{0x77, 0xd5, 0x5f, 0x7d, 0xfd, 0x77}
	directVoiceTS1  = []byte{0x5d, 0x57, 0x7f, 0x77, 0x57, 0xff}
	directDataTS1   = []byte{0xf7, 0xfd, 0xd5, 0xdd, 0xfd, 0x55}
	directVoiceTS2  = []byte{0x7d, 0xff, 0xd5, 0xf5, 0x5d, 0x5f}
	directDataTS2   = []byte{0xd7, 0x55, 0x7f, 0x5f, 0xf7, 0xf5}
	SyncPatternName = map[uint16]string{
		SyncPatternBSSourcedVoice: "bs sourced voice",
		SyncPatternBSSourcedData:  "bs sourced data",
		SyncPatternMSSourcedVoice: "ms sourced voice",
		SyncPatternMSSourcedData:  "ms sourced data",
		SyncPatternMSSourcedRC:    "ms sourced rc",
		SyncPatternDirectVoiceTS1: "direct voice ts1",
		SyncPatternDirectDataTS1:  "direct data ts1",
		SyncPatternDirectVoiceTS2: "direct voice ts2",
		SyncPatternDirectDataTS2:  "direct data ts2",
		SyncPatternUnknown:        "unknown",
	}
)

func ExtractSyncBits(payload bit.Bits) bit.Bits {
	return payload[108:156]
}

func SyncPattern(bits bit.Bits) uint16 {
	var b = bits.Bytes()
	switch {
	case bytes.Equal(b, bsSourcedVoice):
		return SyncPatternBSSourcedVoice
	case bytes.Equal(b, bsSourcedData):
		return SyncPatternBSSourcedData
	case bytes.Equal(b, msSourcedVoice):
		return SyncPatternMSSourcedVoice
	case bytes.Equal(b, msSourcedData):
		return SyncPatternMSSourcedData
	case bytes.Equal(b, msSourcedRC):
		return SyncPatternMSSourcedRC
	case bytes.Equal(b, directVoiceTS1):
		return SyncPatternDirectVoiceTS1
	case bytes.Equal(b, directDataTS1):
		return SyncPatternDirectDataTS1
	case bytes.Equal(b, directVoiceTS2):
		return SyncPatternDirectVoiceTS2
	case bytes.Equal(b, directDataTS2):
		return SyncPatternDirectDataTS2
	default:
		return SyncPatternUnknown
	}
}
