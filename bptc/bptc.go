package bptc

import (
	"fmt"

	"github.com/tehmaze/go-dmr"
	"github.com/tehmaze/go-dmr/fec"
)

func Process(info []byte, payload []byte) error {
	if len(info) < 196 {
		return fmt.Errorf("bptc: info size %d too small, need at least 196 bits", len(info))
	}
	if len(payload) < 12 {
		return fmt.Errorf("bptc: payload size %d too small, need at least 12 bytes", len(payload))
	}

	var (
		i, j, k   uint32
		datafr    = make([]byte, 196)
		extracted = make([]byte, 96)
	)

	// Deinterleave bits
	for i = 1; i < 197; i++ {
		datafr[i-1] = info[((i * 181) % 196)]
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

	copy(payload, dmr.BitsToBytes(extracted))

	return nil
}
