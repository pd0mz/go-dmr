package bit

type Bit byte

func (b *Bit) Flip() {
	(*b) ^= 0x01
}

type Bits []Bit

func toBits(b byte) Bits {
	var o = make(Bits, 8)
	for bit, mask := 0, byte(128); bit < 8; bit, mask = bit+1, mask>>1 {
		if b&mask != 0 {
			o[bit] = 1
		}
	}
	return o
}

func NewBits(bytes []byte) Bits {
	var l = len(bytes)
	var o = make(Bits, 0)
	for i := 0; i < l; i++ {
		o = append(o, toBits(bytes[i])...)
	}
	return o
}

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

func (bits *Bits) String() string {
	var s = ""
	for _, b := range *bits {
		if b == 0x01 {
			s += "1"
		} else {
			s += "0"
		}
	}
	return s
}
