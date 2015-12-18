package bptc

import (
	"fmt"
	"os"

	"github.com/pd0mz/go-dmr"
	"github.com/pd0mz/go-dmr/fec"
)

// deinterleave matrix
var dm = [256]uint8{}
var debug bool

func init() {
	var i uint32
	for i = 0; i < 0x100; i++ {
		dm[i] = uint8((i * 181) % 196)
	}

	debug = os.Getenv("DEBUG_DMR_BPTC") != ""
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
		i, j, k   uint32
		datafr    = make([]byte, 196)
		extracted = make([]byte, 96)
	)

	// Deinterleave bits
	for i = 1; i < 197; i++ {
		datafr[i-1] = info[dm[i]]
	}

	if debug {
		dump(datafr)
	}

	// Zero reserved bits
	for i = 0; i < 3; i++ {
		datafr[0*15+i] = 0
	}

	for i = 0; i < 15; i++ {
		var codeword uint32
		for j = 0; j < 13; j++ {
			codeword <<= 1
			codeword |= uint32(datafr[j*15+i])
		}

		fec.Hamming15_11_3_Correct(&codeword)
		codeword &= 0x01ff
		for j = 0; j < 9; j++ {
			datafr[j*15+i] = byte((codeword >> (8 - j)) & 1)
		}
	}
	for j = 0; j < 9; j++ {
		var codeword uint32
		for i = 0; i < 15; i++ {
			codeword <<= 1
			codeword |= uint32(datafr[j*15+i])
		}
		fec.Hamming15_11_3_Correct(&codeword)
		for i = 0; i < 11; i++ {
			datafr[j*15+10-i] = byte((codeword >> i) & 1)
		}
	}

	// Extract data bits
	for i, k = 3, 0; i < 11; i, k = i+1, k+1 {
		extracted[k] = datafr[0*15+i]
	}
	for j = 1; j < 9; j++ {
		for i = 0; i < 11; i, k = i+1, k+1 {
			extracted[k] = datafr[j*15+i]
		}
	}

	copy(data, dmr.BitsToBytes(extracted))

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
		i, j, k   uint32
		datafr    = make([]byte, 196)
		extracted = make([]byte, 96)
	)

	copy(extracted, dmr.BytesToBits(data))

	for i = 0; i < 9; i++ {
		if i == 0 {
			for j = 3; j < 11; j++ {
				datafr[j+1] = extracted[k]
				k++
			}
		} else {
			for j = 0; j < 11; j++ {
				datafr[j+i*15+1] = extracted[k]
				k++
			}
		}

		datafr[i*15+11+1] = 8
		datafr[i*15+12+1] = 8
		datafr[i*15+13+1] = 8
		datafr[i*15+14+1] = 8
	}

	// Interleave bits
	for i = 1; i < 197; i++ {
		info[dm[i]] = datafr[i-1]
	}

	if debug {
		dump(info)
	}

	return nil
}
