// Package quadres_16_7 implements the quadratic residue (16, 7, 6) parity check.
package quadres_16_7

import "bytes"

var (
	validDataParities = [128][]byte{}
)

type Codeword struct {
	Data   []byte
	Parity []byte
}

func NewCodeword(bits []byte) *Codeword {
	if len(bits) < 16 {
		return nil
	}

	return &Codeword{
		Data:   bits[:7],
		Parity: bits[7:16],
	}
}

func ParityBits(bits []byte) []byte {
	parity := make([]byte, 9)
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

func Check(bits []byte) bool {
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

	return bytes.Equal(codeword.Parity, validDataParities[dataval])
}

func toBits(b byte) []byte {
	var o = make([]byte, 8)
	for bit, mask := 0, byte(128); bit < 8; bit, mask = bit+1, mask>>1 {
		if b&mask != 0 {
			o[bit] = 1
		}
	}
	return o
}

func init() {
	for i := byte(0); i < 128; i++ {
		bits := toBits(i)
		validDataParities[i] = ParityBits(bits)
	}
}
