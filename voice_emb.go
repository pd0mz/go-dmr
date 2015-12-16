package dmr

import (
	"errors"
	"fmt"

	"github.com/tehmaze/go-dmr/crc/quadres_16_7"
)

// EMB LCSS fragments
const (
	SingleFragment uint8 = iota
	FirstFragment
	LastFragment
	Continuation
)

type EMB struct {
	ColorCode uint8
	LCSS      uint8
}

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
