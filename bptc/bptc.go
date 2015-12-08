// Package bptc implements the BPTC(196, 96) Block Product Turbo Code
package bptc

import (
	"errors"
	"fmt"
	"pd0mz/go-dmr/bit"
)

type vector [4]bit.Bit

// hamming(15, 11, 3) checking of a matrix row (15 total bits, 11 data bits,
// min. distance: 3) See page 135 of the DMR Air Interface protocol
// specification for the generator matrix.  A generator matrix looks like this:
// G = [Ik | P]. The parity check matrix is: H = [-P^T|In-k] In binary codes,
// then -P = P, so the negation is unnecessary. We can get the parity check
// matrix only by transposing the generator matrix. We then take a data row,
// and multiply it with each row of the parity check matrix, then xor each
// resulting row bits together with the corresponding parity check bit. The xor
// result (error vector) should be 0, if it's not, it can be used to determine
// the location of the erroneous bit using the generator matrix (P).
func hamming_15_11_3_parity(data bit.Bits, errorVector *vector) {
	if data == nil || len(data) < 11 || errorVector == nil {
		return
	}

	var e = *errorVector
	e[0] = (data[0] ^ data[1] ^ data[2] ^ data[3] ^ data[5] ^ data[7] ^ data[8])
	e[1] = (data[1] ^ data[2] ^ data[3] ^ data[4] ^ data[6] ^ data[8] ^ data[9])
	e[2] = (data[2] ^ data[3] ^ data[4] ^ data[5] ^ data[7] ^ data[9] ^ data[10])
	e[3] = (data[0] ^ data[1] ^ data[2] ^ data[4] ^ data[6] ^ data[7] ^ data[10])
}

func hamming_15_11_3_check(data bit.Bits, errorVector *vector) bool {
	if data == nil || len(data) < 15 || errorVector == nil {
		return false
	}

	hamming_15_11_3_parity(data, errorVector)

	var e = *errorVector
	e[0] ^= data[11]
	e[1] ^= data[12]
	e[2] ^= data[13]
	e[3] ^= data[14]

	return e[0] == 0 && e[1] == 0 && e[2] == 0 && e[3] == 0
}

func hamming_13_9_3_parity(data bit.Bits, errorVector *vector) {
	if data == nil || len(data) < 9 || errorVector == nil {
		return
	}

	var e = *errorVector
	e[0] = (data[0] ^ data[1] ^ data[3] ^ data[5] ^ data[6])
	e[1] = (data[0] ^ data[1] ^ data[2] ^ data[4] ^ data[6] ^ data[7])
	e[2] = (data[0] ^ data[1] ^ data[2] ^ data[3] ^ data[5] ^ data[7] ^ data[8])
	e[3] = (data[0] ^ data[2] ^ data[4] ^ data[5] ^ data[8])
}

// hamming(13, 9, 3) checking of a matrix column (13 total bits, 9 data bits,
// min. distance: 3)
func hamming_13_9_3_check(data bit.Bits, errorVector *vector) bool {
	if data == nil || len(data) < 13 || errorVector == nil {
		return false
	}

	hamming_13_9_3_parity(data, errorVector)

	var e = *errorVector
	e[0] ^= data[9]
	e[1] ^= data[10]
	e[2] ^= data[11]
	e[3] ^= data[12]

	return e[0] == 0 && e[1] == 0 && e[2] == 0 && e[3] == 0
}

var hamming_15_11_generator_matrix = bit.Bits{
	1, 0, 0, 1,
	1, 1, 0, 1,
	1, 1, 1, 1,
	1, 1, 1, 0,
	0, 1, 1, 1,
	1, 0, 1, 0,
	0, 1, 0, 1,
	1, 0, 1, 1,
	1, 1, 0, 0,
	0, 1, 1, 0,
	0, 0, 1, 1,

	1, 0, 0, 0, // These are used to determine errors in the hamming checksum bits.
	0, 1, 0, 0,
	0, 0, 1, 0,
	0, 0, 0, 1,
}

func hamming_15_11_3_error_position(errorVector *vector) int {
	if errorVector == nil {
		return -1
	}
	var e = *errorVector
	for row := 0; row < 15; row++ {
		if hamming_15_11_generator_matrix[row*4] == e[0] &&
			hamming_15_11_generator_matrix[row*4+1] == e[1] &&
			hamming_15_11_generator_matrix[row*4+2] == e[2] &&
			hamming_15_11_generator_matrix[row*4+3] == e[3] {
			return row
		}
	}
	return -1
}

var hamming_13_9_generator_matrix = bit.Bits{
	1, 1, 1, 1,
	1, 1, 1, 0,
	0, 1, 1, 1,
	0, 1, 1, 1,
	0, 1, 0, 1,
	1, 0, 1, 1,
	1, 1, 0, 0,
	0, 1, 1, 0,
	0, 0, 1, 1,

	1, 0, 0, 0, // These are used to determine errors in the hamming checksum bits.
	0, 1, 0, 0,
	0, 0, 1, 0,
	0, 0, 0, 1,
}

func hamming_13_9_3_error_position(errorVector *vector) int {
	if errorVector == nil {
		return -1
	}
	var e = *errorVector
	for row := 0; row < 13; row++ {
		if hamming_13_9_generator_matrix[row*4] == e[0] &&
			hamming_13_9_generator_matrix[row*4+1] == e[1] &&
			hamming_13_9_generator_matrix[row*4+2] == e[2] &&
			hamming_13_9_generator_matrix[row*4+3] == e[3] {
			return row
		}
	}
	return -1
}

func Dump(bits bit.Bits) {
	if len(bits) != 196 {
		return
	}

	var row, col int

	fmt.Println("    BPTC(196, 96) matrix:")
	for row = 0; row < 13; row++ {
		for col = 0; col < 11; col++ {
			// +1 because the first bit is R(3) and it's not used
			// so we can ignore that.
			fmt.Printf("      #%.2u ", bits[col+row*15+1])
		}
		fmt.Print(" ")
		for ; col < 15; col++ {
			// +1 because the first bit is R(3) and it's not used
			// so we can ignore that.
			fmt.Printf("%u", bits[col+row*15+1])
		}
		fmt.Println("")
		if row == 8 {
			fmt.Println("")
		}
	}
}

func CheckAndRepair(bits bit.Bits) (bool, error) {
	if bits == nil || len(bits) != 196 {
		return false, errors.New("expected 196 input bits")
	}

	var (
		cb          = make([]bit.Bit, 13)
		errorVector = vector{}
	)
	for col := 0; col < 15; col++ {
		for row := 0; row < 13; row++ {
			// +1 because the first bit is R(3) and it's not used so we can ignore that.
			cb[row] = bits[col+row*15+1]
		}

		if !hamming_13_9_3_check(cb, &errorVector) {
			wrong := hamming_13_9_3_error_position(&errorVector)
			if wrong < 0 {
				return false, fmt.Errorf("dmr/bptc(196, 96): hamming(13, 9) check error in column #%u, can't repair", col)
			}

			// Fix bit error
			bits[col+wrong*15+1] ^= 1
			for row := 0; row < 13; row++ {
				// +1 because the first bit is R(3) and it's not used so we can ignore that.
				cb[row] = bits[col+row*15+1]
			}

			if !hamming_13_9_3_check(cb, &errorVector) {
				return false, fmt.Errorf("dmr/bptc(196, 96): hamming(13, 9) check error in column #%u, couldn't repair", col)
			}
		}
	}

	for row := 0; row < 9; row++ {
		// +1 because the first bit is R(3) and it's not used so we can ignore that.
		if !hamming_15_11_3_check(bits[row*15+1:], &errorVector) {
			wrong := hamming_15_11_3_error_position(&errorVector)
			if wrong < 0 {
				return false, fmt.Errorf("dmr/bptc(196, 96): hamming(15, 11) check error in row #%u, can't repair", row)
			}

			// Fix bit error
			bits[row*15+wrong+1] ^= 1
			if !hamming_15_11_3_check(bits[row*15+1:], &errorVector) {
				return false, fmt.Errorf("dmr/bptc (196,96): hamming(15,11) check error, couldn't repair row #%u", row)
			}
		}
	}

	return true, nil
}

// Extract the data bits from the given deinterleaved info bits array (discards BPTC bits).
func Extract(bits bit.Bits) bit.Bits {
	var e = make([]bit.Bit, 96)
	copy(e[0:8], bits[4:12])
	copy(e[8:19], bits[16:27])
	copy(e[19:30], bits[31:42])
	copy(e[30:41], bits[46:57])
	copy(e[41:52], bits[61:72])
	copy(e[52:63], bits[76:87])
	copy(e[63:74], bits[91:102])
	copy(e[74:85], bits[106:117])
	copy(e[85:96], bits[121:132])
	return e
}

// New BPTC(196, 96) payload from 96 data bits.
func New(bits bit.Bits) bit.Bits {
	var (
		dbp         int
		errorVector = vector{}
		p           = make([]bit.Bit, 196)
	)

	for row := 0; row < 9; row++ {
		if row == 0 {
			for col := 3; col < 11; col++ {
				// +1 because the first bit is R(3) and it's not used so we can ignore that.
				p[col+1] = bits[dbp]
				dbp++
			}
		} else {
			for col := 0; col < 11; col++ {
				// +1 because the first bit is R(3) and it's not used so we can ignore that.
				p[col+row*15+1] = bits[dbp]
				dbp++
			}
		}
		// +1 because the first bit is R(3) and it's not used so we can ignore that.
		hamming_15_11_3_parity(bits[row*15+1:], &errorVector)
		bits[row*15+11+1] = errorVector[0]
		bits[row*15+12+1] = errorVector[1]
		bits[row*15+13+1] = errorVector[2]
		bits[row*15+14+1] = errorVector[3]
	}

	for col := 0; col < 15; col++ {
		var cb = make([]bit.Bit, 9)
		for row := 0; row < 9; row++ {
			cb[row] = bits[col+row*15+1]
		}
		hamming_13_9_3_parity(cb, &errorVector)
		bits[col+135+1] = errorVector[0]
		bits[col+135+15+1] = errorVector[1]
		bits[col+135+30+1] = errorVector[2]
		bits[col+135+45+1] = errorVector[3]
	}

	return p
}
