// Package dmr contains generic message structures for the Digital Mobile Radio standard
package dmr

import (
	"fmt"

	"github.com/pd0mz/go-dmr/bit"
)

// Burst contains data from a single burst, see 4.2.2 Burst and frame structure
type Burst struct {
	bits bit.Bits
}

func NewBurst(raw []byte) (*Burst, error) {
	if len(raw)*8 != PayloadBits {
		return nil, fmt.Errorf("dmr: expected %d bits, got %d", PayloadBits, len(raw)*8)
	}
	b := &Burst{}
	b.bits = bit.NewBits(raw)
	return b, nil
}
