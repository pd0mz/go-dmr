package bptc

import (
	"fmt"
	"os"

	"github.com/pd0mz/go-dmr"
)

var (
	debug bool

	// deinterleave matrix
	dm = [256]uint8{}
)

func init() {
	debug = os.Getenv("DEBUG_DMR_BPTC") != ""

	var i uint32
	for i = 0; i < 0x100; i++ {
		dm[i] = uint8((i * 181) % 196)
	}
}

func dump(bits []byte) {
	for row := 0; row < 13; row++ {
		if row == 0 {
			fmt.Printf("col #    ")
			for col := 0; col < 15; col++ {
				fmt.Printf("%02d ", col+1)
				if col == 10 {
					fmt.Print("| ")
				}
			}
			fmt.Println("")
		}
		if row == 9 {
			fmt.Println("          -------------------------------   ------------")
		}
		for col := 0; col < 15; col++ {
			if col == 0 {
				fmt.Printf("row #%02d: ", row+1)
			}
			fmt.Printf(" %d ", bits[col+row*15+1])
			if col == 10 {
				fmt.Print("| ")
			}
		}
		fmt.Println("")
	}
}

func Decode(info, data []byte) error {
	if len(info) < 196 {
		return fmt.Errorf("bptc: info size %d too small, need at least 196 bits", len(info))
	}
	if len(data) < 12 {
		return fmt.Errorf("bptc: data size %d too small, need at least 12 bytes", len(data))
	}

	var (
		i, j, k uint32
		bits    = make([]byte, 196)
		temp    = make([]byte, 196)
	)

	// Deinterleave
	for i = 1; i < 197; i++ {
		bits[i-1] = info[dm[i]]
	}

	if debug {
		dump(bits)
	}

	// Hamming checks
	if err := hamming_check(bits); err != nil {
		return err
	}

	// Extract data bits
	for i, k = 3, 0; i < 11; i, k = i+1, k+1 {
		temp[k] = bits[0*15+i]
	}
	for j = 1; j < 9; j++ {
		for i = 0; i < 11; i, k = i+1, k+1 {
			temp[k] = bits[j*15+i]
		}
	}

	copy(data, dmr.BitsToBytes(temp))
	return nil
}

func Encode(data, info []byte) error {
	if len(data) < 12 {
		return fmt.Errorf("bptc: data size %d too small, need at least 12 bytes", len(data))
	}
	if len(info) < 196 {
		return fmt.Errorf("bptc: info size %d too small, need at least 196 bits", len(info))
	}

	var (
		bits = dmr.BytesToBits(data)
		temp = make([]byte, 196)
		errs = make([]byte, 4)
		cols = make([]byte, 13)
	)

	var c, r, k uint32
	for r = 0; r < 9; r++ {
		if r == 0 {
			for c = 3; c < 11; c, k = c+1, k+1 {
				temp[c] = bits[k]
			}
		} else {
			for c = 0; c < 11; c, k = c+1, k+1 {
				temp[c+r*15] = bits[k]
			}
		}

		hamming_15_11_3_parity(temp[r*15:], errs)
		temp[r*15+11] = errs[0]
		temp[r*15+12] = errs[1]
		temp[r*15+13] = errs[2]
		temp[r*15+14] = errs[3]
	}
	for c = 0; c < 15; c++ {
		for r = 0; r < 9; r++ {
			cols[r] = temp[c+r*15+1]
		}

		hamming_13_9_3_parity(cols, errs)
		temp[c+135+1] = errs[0]
		temp[c+135+15+1] = errs[1]
		temp[c+135+30+1] = errs[2]
		temp[c+135+45+1] = errs[3]
	}

	if debug {
		dump(temp)
	}

	// Interleave
	for k = 1; k < 197; k++ {
		info[dm[k]] = temp[k-1]
	}

	return nil
}

// Hamming(13, 9, 3) check
func hamming_13_9_3_parity(bits, errs []byte) bool {
	errs[0] = bits[0] ^ bits[1] ^ bits[3] ^ bits[5] ^ bits[6]
	errs[1] = bits[0] ^ bits[1] ^ bits[2] ^ bits[4] ^ bits[6] ^ bits[7]
	errs[2] = bits[0] ^ bits[1] ^ bits[2] ^ bits[3] ^ bits[5] ^ bits[7] ^ bits[8]
	errs[3] = bits[0] ^ bits[2] ^ bits[4] ^ bits[5] ^ bits[8]
	return (errs[0] == bits[9]) && (errs[1] == bits[10]) && (errs[2] == bits[11]) && (errs[3] == bits[12])
}

// Hamming(15, 11, 3) check
func hamming_15_11_3_parity(bits, errs []byte) bool {
	errs[0] = bits[0] ^ bits[1] ^ bits[2] ^ bits[3] ^ bits[5] ^ bits[7] ^ bits[8]
	errs[1] = bits[1] ^ bits[2] ^ bits[3] ^ bits[4] ^ bits[6] ^ bits[8] ^ bits[9]
	errs[2] = bits[2] ^ bits[3] ^ bits[4] ^ bits[5] ^ bits[7] ^ bits[9] ^ bits[10]
	errs[3] = bits[0] ^ bits[1] ^ bits[2] ^ bits[4] ^ bits[6] ^ bits[7] ^ bits[10]
	return (errs[0] == bits[11]) && (errs[1] == bits[12]) && (errs[2] == bits[13]) && (errs[3] == bits[14])
}

// hamming_check checks each row with a Hamming(15,11,3) code and each column with Hamming(13, 9, 3)
func hamming_check(bits []byte) error {
	var (
		c, r, k uint32
		row     = make([]byte, 15)
		col     = make([]byte, 13)
		errs    = make([]byte, 4)
	)

	// Run through each of the 9 rows containing data
	for r = 0; r < 9; r++ {
		k = r*15 + 1
		for a := 0; a < 15; a++ {
			row[a] = bits[k]
		}
		if !hamming_15_11_3_parity(row, errs) {
			return fmt.Errorf("hamming(15, 11, 3) check failed on row #%d", r)
		}
	}

	// Run through each of the 15 columns
	for c = 0; c < 15; c++ {
		k = c + 1
		for a := 0; a < 13; a, k = a+1, k+15 {
			col[a] = bits[k]
		}
		if !hamming_13_9_3_parity(col, errs) {
			return fmt.Errorf("hamming(13, 9, 3) check failed on col #%d", c)
		}
	}

	return nil
}
