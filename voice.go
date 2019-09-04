package dmr

import (
	"errors"
	"fmt"

	"github.com/pd0mz/go-dmr/crc/quadres_16_7"
)

// EMB LCSS fragments.
const (
	SingleFragment uint8 = iota
	FirstFragment
	LastFragment
	Continuation
)

// LCSSName is a map of LCSS fragment type to string.
var LCSSName = map[uint8]string{
	SingleFragment: "single fragment",
	FirstFragment:  "first fragment",
	LastFragment:   "last fragment",
	Continuation:   "continuation",
}

// EMB contains embedded signalling.
type EMB struct {
	ColorCode uint8
	LCSS      uint8
}

func (emb *EMB) String() string {
	return fmt.Sprintf("color code %d, %s (%d)", emb.ColorCode, LCSSName[emb.LCSS], emb.LCSS)
}

// ParseEMB parses embedded signalling
func ParseEMB(bits []byte) (*EMB, error) {
	if len(bits) != EMBBits {
		return nil, fmt.Errorf("dmr/emb: expected %d bits, got %d", EMBBits, len(bits))
	}

	if !quadres_16_7.Check(bits) {
		return nil, errors.New("dmr/emb: checksum error")
	}

	if bits[4] != 0 {
		return nil, errors.New("dmr/emb: pi is not 0")
	}

	return &EMB{
		ColorCode: uint8(bits[0])<<3 | uint8(bits[1])<<2 | uint8(bits[2])<<1 | uint8(bits[3]),
		LCSS:      uint8(bits[5])<<1 | uint8(bits[6]),
	}, nil
}

// ParseEMBBitsFromSync extracts the embedded signalling bits from the SYNC bits.
func ParseEMBBitsFromSync(sync []byte) ([]byte, error) {
	if sync == nil {
		return nil, errors.New("dmr/emb from sync: bits can't be nil")
	}
	if len(sync) != 48 {
		return nil, fmt.Errorf("dmr/emb from sync: expected 48 sync bits, got %d", len(sync))
	}

	var bits = make([]byte, 16)
	copy(bits[:8], sync[:8])
	copy(bits[8:], sync[8+32:])
	return bits, nil
}

// ParseEmbeddedSignallingLCFromSyncBits extracts the embedded signalling LC from the SYNC bits.
func ParseEmbeddedSignallingLCFromSyncBits(sync []byte) ([]byte, error) {
	if sync == nil {
		return nil, errors.New("dmr/emb lc from sync: bits can't be nil")
	}
	if len(sync) != 48 {
		return nil, fmt.Errorf("dmr/emb lc from sync: expected 48 sync bits, got %d", len(sync))
	}

	var bits = make([]byte, 32)
	copy(bits, sync[8:40])
	return bits, nil
}

// EmbeddedSignallingLC contains the embedded signalling LC and checksum.
type EmbeddedSignallingLC struct {
	Bits     []byte
	Checksum []byte
}

// Check verifies the checksum in the embedded signalling LC.
func (eslc *EmbeddedSignallingLC) Check() bool {
	var checksum uint8
	checksum |= eslc.Checksum[0] << 4
	checksum |= eslc.Checksum[1] << 3
	checksum |= eslc.Checksum[2] << 2
	checksum |= eslc.Checksum[3] << 1
	checksum |= eslc.Checksum[4] << 0

	var data = BitsToBytes(eslc.Bits)
	var verify uint16
	for _, b := range data {
		verify += uint16(b)
	}

	var calculated = uint8(verify % 31)
	return calculated == checksum
}

// Interleave packs the embedded signalling LC to interleaved bits.
func (eslc *EmbeddedSignallingLC) Interleave() []byte {
	var bits = make([]byte, 77)
	var j int
	for i := range bits {
		switch i {
		case 32:
			bits[i] = eslc.Checksum[0]
			break
		case 43:
			bits[i] = eslc.Checksum[1]
			break
		case 54:
			bits[i] = eslc.Checksum[2]
			break
		case 65:
			bits[i] = eslc.Checksum[3]
			break
		case 76:
			bits[i] = eslc.Checksum[4]
			break
		default:
			bits[i] = eslc.Bits[j]
			j++
		}
	}

	return bits
}

// DeinterleaveEmbeddedSignallingLC deinterleaves the embedded signalling LC bits.
func DeinterleaveEmbeddedSignallingLC(bits []byte) (*EmbeddedSignallingLC, error) {
	if bits == nil {
		return nil, errors.New("dmr/emb lc deinterleave: bits can't be nil")
	}
	if len(bits) != 77 {
		return nil, fmt.Errorf("dmr/emb lc deinterleave: expected 77 bits, got %d", len(bits))
	}

	var eslc = &EmbeddedSignallingLC{
		Bits:     make([]byte, 72),
		Checksum: make([]byte, 5),
	}
	var j int
	for i, b := range bits {
		switch i {
		case 32:
			eslc.Checksum[0] = b
			break
		case 43:
			eslc.Checksum[1] = b
			break
		case 54:
			eslc.Checksum[2] = b
			break
		case 65:
			eslc.Checksum[3] = b
			break
		case 76:
			eslc.Checksum[4] = b
			break
		default:
			eslc.Bits[j] = b
			j++
		}
	}

	return eslc, nil
}
