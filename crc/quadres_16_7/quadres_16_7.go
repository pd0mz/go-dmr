// Package quadres_16_7 implements the quadratic residue (16, 7, 6) parity check.
package quadres_16_7

import "github.com/tehmaze/go-dmr/bit"

var (
	validDataParities = [128]bit.Bits{}
)

type Codeword struct {
	Data   bit.Bits
	Parity bit.Bits
}

func NewCodeword(bits bit.Bits) *Codeword {
	if len(bits) < 16 {
		return nil
	}

	return &Codeword{
		Data:   bits[:7],
		Parity: bits[7:16],
	}
}

func ParityBits(bits bit.Bits) bit.Bits {
	parity := make(bit.Bits, 9)
	// Multiplying the generator matrix with the given data bits.
	// See DMR AI spec. page 134.
	parity[0] = bits[1] ^ bits[2] ^ bits[3] ^ bits[4]
	parity[1] = bits[2] ^ bits[3] ^ bits[4] ^ bits[5]
	parity[2] = bits[0] ^ bits[3] ^ bits[4] ^ bits[5] ^ bits[6]
	parity[3] = bits[2] ^ bits[3] ^ bits[5] ^ bits[6]
	parity[4] = bits[1] ^ bits[2] ^ bits[6]
	parity[5] = bits[0] ^ bits[1] ^ bits[4]
	parity[6] = bits[0] ^ bits[1] ^ bits[2] ^ bits[5]
	parity[7] = bits[0] ^ bits[1] ^ bits[2] ^ bits[3] ^ bits[6]
	parity[8] = bits[0] ^ bits[2] ^ bits[4] ^ bits[5] ^ bits[6]
	return parity
}

func Check(bits bit.Bits) bool {
	codeword := NewCodeword(bits)
	if codeword == nil {
		return false
	}

	var dataval uint8
	for col := uint8(0); col < 7; col++ {
		if codeword.Data[col] == 1 {
			dataval |= (1 << (7 - col))
		}
	}

	return codeword.Parity.Equal(validDataParities[dataval])
}

func init() {
	for i := byte(0); i < 128; i++ {
		bits := bit.NewBits([]byte{i})
		validDataParities[i] = ParityBits(bits)
	}
}
