// Package vbptc implements the Variable length BPTC for embedded signalling
package vbptc

import (
	"errors"
	"fmt"
)

var (
	// See page 136 of the DMR AI. spec. for the generator matrix.
	hamming_16_11_generator_matrix = []byte{
		1, 0, 0, 1, 1,
		1, 1, 0, 1, 0,
		1, 1, 1, 1, 1,
		1, 1, 1, 0, 0,
		0, 1, 1, 1, 0,
		1, 0, 1, 0, 1,
		0, 1, 0, 1, 1,
		1, 0, 1, 1, 0,
		1, 1, 0, 0, 1,
		0, 1, 1, 0, 1,
		0, 0, 1, 1, 1,
		// These are used to determine errors in the Hamming checksum bits.
		1, 0, 0, 0, 0,
		0, 1, 0, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 0, 1, 0,
		0, 0, 0, 0, 1,
	}
)

type VBPTC struct {
	matrix       []byte
	row, col     uint8
	expectedRows uint8
}

func New(expectedRows uint8) *VBPTC {
	return &VBPTC{
		matrix:       make([]byte, int(expectedRows)*16),
		expectedRows: expectedRows,
	}
}

func (v *VBPTC) freeSpace() int {
	var size = int(v.expectedRows) * 16
	var used = int(v.expectedRows)*int(v.col) + int(v.row)
	return size - used
}

// AddBurst adds the embedded signalling data to the matrix.
func (v *VBPTC) AddBurst(bits []byte) error {
	if v.matrix == nil {
		return errors.New("vbptc: matrix can't be nil")
	}
	var free = v.freeSpace()
	if free == 0 {
		return errors.New("vbptc: no free space in matrix")
	}

	var adds = len(bits)
	if adds > free {
		adds = free
	}

	for i := 0; i < adds; i++ {
		v.matrix[v.col+v.row*16] = bits[i]
		v.row++
		if v.row == v.expectedRows {
			v.col++
			v.row = 0
		}
	}

	return nil
}

// CheckAndRepair checks data for errors and tries to repair them
func (v *VBPTC) CheckAndRepair() error {
	if v.matrix == nil || v.expectedRows < 2 {
		return fmt.Errorf("vbptc: no data")
	}

	var (
		row, col uint8
		errs     = make([]byte, 5)
	)

	// -1 because the last row contains only single parity check bits
	for row = 0; row < v.expectedRows-1; row++ {
		if !checkRow(v.matrix[row*16:], errs) {
			// If the Hamming(16, 11, 4) column check failed, see if we can find
			// the bit error location.
			pos, found := findPosition(errs)
			if !found {
				return fmt.Errorf("vbptc: hamming(16,11) check error, can't repair row #%d", row)
			}

			// Flip wrong bit
			v.matrix[row*16+pos] ^= 1
			if !checkRow(v.matrix[row*16:], errs) {
				return fmt.Errorf("vbptc: hamming(16,11) check error, couldn't repair row #%d", row)
			}
		}
	}

	for col = 0; col < 16; col++ {
		var parity uint8
		for row = 0; row < v.expectedRows-1; row++ {
			parity = (parity + v.matrix[row*16+col]) % 2
		}
		if parity != v.matrix[(v.expectedRows-1)*16+col] {
			return fmt.Errorf("vbptc: parity check error in column #%d", col)
		}
	}

	return nil
}

// Clear resets the variable BPTC matrix and cursor position
func (v *VBPTC) Clear() {
	v.row = 0
	v.col = 0
	v.matrix = make([]byte, int(v.expectedRows)*16)
}

// GetData extracts data bits (discarding Hamming (16,11) and parity check bits) from the vbptc matrix.
func (v *VBPTC) GetData(bits []byte) error {
	if v.matrix == nil || v.expectedRows == 0 {
		return errors.New("vbptc: no data in matrix")
	}
	if bits == nil {
		return errors.New("vbptc: bits can't be nil")
	}
	if len(bits) < 77 {
		return fmt.Errorf("vbptc: need at least 77 bits buffer, got %d", len(bits))
	}

	var row, col uint8
	for row = 0; row < v.expectedRows-1; row++ {
		for col = 0; col < 11; col++ {
			bits[row*11+col] = v.matrix[row*16+col]
		}
	}

	return nil
}

func checkRow(bits, errs []byte) bool {
	if bits == nil || errs == nil {
		return false
	}

	getParity(bits, errs)
	errs[0] ^= bits[11]
	errs[1] ^= bits[12]
	errs[2] ^= bits[13]
	errs[3] ^= bits[14]
	errs[4] ^= bits[15]

	return errs[0] == 0 && errs[1] == 0 && errs[2] == 0 && errs[3] == 0 && errs[4] == 0
}

func findPosition(errs []byte) (uint8, bool) {
	for row := uint8(0); row < 16; row++ {
		var found = true
		switch {
		case hamming_16_11_generator_matrix[row*5] != errs[0]:
			found = false
			break
		case hamming_16_11_generator_matrix[row*5+1] != errs[1]:
			found = false
			break
		case hamming_16_11_generator_matrix[row*5+2] != errs[2]:
			found = false
			break
		case hamming_16_11_generator_matrix[row*5+3] != errs[3]:
			found = false
			break
		}
		if found {
			return row, true
		}
	}

	return 0, false
}

func getParity(bits, errs []byte) {
	errs[0] = (bits[0] ^ bits[1] ^ bits[2] ^ bits[3] ^ bits[5] ^ bits[7] ^ bits[8])
	errs[1] = (bits[1] ^ bits[2] ^ bits[3] ^ bits[4] ^ bits[6] ^ bits[8] ^ bits[9])
	errs[2] = (bits[2] ^ bits[3] ^ bits[4] ^ bits[5] ^ bits[7] ^ bits[9] ^ bits[10])
	errs[3] = (bits[0] ^ bits[1] ^ bits[2] ^ bits[4] ^ bits[6] ^ bits[7] ^ bits[10])
	errs[4] = (bits[0] ^ bits[2] ^ bits[5] ^ bits[6] ^ bits[8] ^ bits[9] ^ bits[10])
}
