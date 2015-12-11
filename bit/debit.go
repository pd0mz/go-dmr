package bit

type Debit uint8

type Debits []Debit

func toDebits(b byte) Debits {
	var o = make(Debits, 4)
	for bit, mask := 0, byte(128); bit < 8; bit, mask = bit+2, mask>>2 {
		o[bit/2] = Debit((b >> mask) & 3)
	}
	return o
}

func NewDebits(bytes []byte) Debits {
	var l = len(bytes)
	var o = make(Debits, 0)
	for i := 0; i < l; i++ {
		o = append(o, toDebits(bytes[i])...)
	}
	return o
}
