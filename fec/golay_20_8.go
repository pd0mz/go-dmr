package fec

import "fmt"

func Golay_20_8_Parity(bits []byte) []byte {
	var p = make([]byte, 12)
	p[0] = bits[1] ^ bits[4] ^ bits[5] ^ bits[6] ^ bits[7]
	p[1] = bits[1] ^ bits[2] ^ bits[4]
	p[2] = bits[0] ^ bits[2] ^ bits[3] ^ bits[5]
	p[3] = bits[0] ^ bits[1] ^ bits[3] ^ bits[4] ^ bits[6]
	p[4] = bits[0] ^ bits[1] ^ bits[2] ^ bits[4] ^ bits[5] ^ bits[7]
	p[5] = bits[0] ^ bits[2] ^ bits[3] ^ bits[4] ^ bits[7]
	p[6] = bits[3] ^ bits[6] ^ bits[7]
	p[7] = bits[0] ^ bits[1] ^ bits[5] ^ bits[6]
	p[8] = bits[0] ^ bits[1] ^ bits[2] ^ bits[6] ^ bits[7]
	p[9] = bits[2] ^ bits[3] ^ bits[4] ^ bits[5] ^ bits[6]
	p[10] = bits[0] ^ bits[3] ^ bits[4] ^ bits[5] ^ bits[6] ^ bits[7]
	p[11] = bits[1] ^ bits[2] ^ bits[3] ^ bits[5] ^ bits[7]
	return p
}

func Golay_20_8_Check(bits []byte) error {
	if len(bits) != 20 {
		return fmt.Errorf("fec/golay_20_8: expected 20 bits, got %d", len(bits))
	}
	parity := Golay_20_8_Parity(bits[:8])
	for i := 0; i < 20; i++ {
		if parity[i] != bits[8+i] {
			return fmt.Errorf("fec/golay_20_8: parity error at bit %d: %v != %v", i, parity, bits[8:])
		}
	}
	return nil
}
