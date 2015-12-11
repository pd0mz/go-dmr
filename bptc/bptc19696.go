package bptc

import (
	"fmt"

	"github.com/pd0mz/go-dmr/bit"
)

func Decode(bits bit.Bits, deinterleave bool) (bit.Bits, error) {
	var debits bit.Bits
	if deinterleave {
		debits = Deinterleave(bits)
	} else {
		debits = bits
	}
	if err := Check(bits); err != nil {
		return nil, err
	}

	return Extract(debits), nil
}

// Deinterleave raw bits
func Deinterleave(r bit.Bits) bit.Bits {
	// The first bit is R(3) which is not used so can be ignored
	var d = make(bit.Bits, 196)
	var i int
	for a := 0; a < 196; a++ {
		i = (a * 181) % 196
		d[a] = r[i]
	}
	return d
}

// Hamming(13, 9, 3) check
func Hamming1393(debits bit.Bits) (bool, bit.Bits) {
	var err = make(bit.Bits, 4)
	err[0] = debits[0] ^ debits[1] ^ debits[3] ^ debits[5] ^ debits[6]
	err[1] = debits[0] ^ debits[1] ^ debits[2] ^ debits[4] ^ debits[6] ^ debits[7]
	err[2] = debits[0] ^ debits[1] ^ debits[2] ^ debits[3] ^ debits[5] ^ debits[7] ^ debits[8]
	err[3] = debits[0] ^ debits[2] ^ debits[4] ^ debits[5] ^ debits[8]
	return (err[0] == debits[9]) && (err[1] == debits[10]) && (err[2] == debits[11]) && (err[3] == debits[12]), err
}

// Hamming(15, 11, 3) check
func Hamming15113(debits bit.Bits) (bool, bit.Bits) {
	var err = make(bit.Bits, 4)
	err[0] = debits[0] ^ debits[1] ^ debits[2] ^ debits[3] ^ debits[5] ^ debits[7] ^ debits[8]
	err[1] = debits[1] ^ debits[2] ^ debits[3] ^ debits[4] ^ debits[6] ^ debits[8] ^ debits[9]
	err[2] = debits[2] ^ debits[3] ^ debits[4] ^ debits[5] ^ debits[7] ^ debits[9] ^ debits[10]
	err[3] = debits[0] ^ debits[1] ^ debits[2] ^ debits[4] ^ debits[6] ^ debits[7] ^ debits[10]
	return (err[0] == debits[11]) && (err[1] == debits[12]) && (err[2] == debits[13]) && (err[3] == debits[14]), err
}

// Check each row with a Hamming (15,11,3) code
func Check(debits bit.Bits) error {
	var (
		row = make(bit.Bits, 15)
		col = make(bit.Bits, 13)
	)

	// Run through each of the 9 rows containing data
	for r := 0; r < 9; r++ {
		p := (r * 15) + 1
		for a := 0; a < 15; a++ {
			row[a] = debits[p]
		}
		if ok, _ := Hamming15113(row); !ok {
			return fmt.Errorf("hamming(15, 11, 3) check failed on row #%d", r)
		}
	}

	// Run through each of the 15 columns
	for c := 0; c < 15; c++ {
		p := c + 1
		for a := 0; a < 13; a++ {
			col[a] = debits[p]
			p += 15
		}
		if ok, _ := Hamming1393(col); !ok {
			return fmt.Errorf("hamming(13, 9, 3) check failed on col #%d", c)
		}
	}

	return nil
}

// Extract the 96 bits of data
func Extract(debits bit.Bits) bit.Bits {
	var (
		out    = make(bit.Bits, 96)
		a, pos int
	)

	for a = 4; a <= 11; a++ {
		out[pos] = debits[a]
		pos++
	}
	for a = 16; a <= 26; a++ {
		out[pos] = debits[a]
		pos++
	}
	for a = 31; a <= 41; a++ {
		out[pos] = debits[a]
		pos++
	}
	for a = 46; a <= 56; a++ {
		out[pos] = debits[a]
		pos++
	}
	for a = 61; a <= 71; a++ {
		out[pos] = debits[a]
		pos++
	}
	for a = 76; a <= 86; a++ {
		out[pos] = debits[a]
		pos++
	}
	for a = 91; a <= 101; a++ {
		out[pos] = debits[a]
		pos++
	}
	for a = 106; a <= 116; a++ {
		out[pos] = debits[a]
		pos++
	}
	for a = 121; a <= 131; a++ {
		out[pos] = debits[a]
		pos++
	}

	return out
}
