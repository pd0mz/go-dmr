package bit

type Bit byte

func (b *Bit) Flip() {
	(*b) ^= 0x01
}

type Bits []Bit

func (bits *Bits) Bytes() []byte {
	var l = len(*bits)
	var o = make([]byte, (l+7)/8)
	for i, b := range *bits {
		if b == 0x01 {
			o[i/8] |= (1 << byte(7-(i%8)))
		}
	}
	return o
}
