package trellis

import (
	"errors"
	"fmt"
	"log"

	"github.com/pd0mz/go-dmr"
)

var (
	// See DMR AI protocol spec. page 130.
	interleaveMatrix = []uint8{
		0, 1, 8, 9, 16, 17, 24, 25, 32, 33, 40, 41, 48, 49, 56, 57, 64, 65, 72, 73, 80, 81, 88, 89, 96, 97,
		2, 3, 10, 11, 18, 19, 26, 27, 34, 35, 42, 43, 50, 51, 58, 59, 66, 67, 74, 75, 82, 83, 90, 91,
		4, 5, 12, 13, 20, 21, 28, 29, 36, 37, 44, 45, 52, 53, 60, 61, 68, 69, 76, 77, 84, 85, 92, 93,
		6, 7, 14, 15, 22, 23, 30, 31, 38, 39, 46, 47, 54, 55, 62, 63, 70, 71, 78, 79, 86, 87, 94, 95,
	}

	// See DMR AI protocol spec. page 129.
	encoderStateTransition = []uint8{
		0, 8, 4, 12, 2, 10, 6, 14,
		4, 12, 2, 10, 6, 14, 0, 8,
		1, 9, 5, 13, 3, 11, 7, 15,
		5, 13, 3, 11, 7, 15, 1, 9,
		3, 11, 7, 15, 1, 9, 5, 13,
		7, 15, 1, 9, 5, 13, 3, 11,
		2, 10, 6, 14, 0, 8, 4, 12,
		6, 14, 0, 8, 4, 12, 2, 10,
	}
)

func init() {
	log.Printf("interleave matrix has %d points\n", len(interleaveMatrix))
}

// Decode is a convenience function that takes 196 Info bits and decodes them to 18 bytes (144 bits) binary using Trellis decoding.
func Decode(bits []byte, bytes []byte) error {
	if bytes == nil {
		return errors.New("trellis: bytes can't be nil")
	}
	if len(bytes) < 18 {
		return fmt.Errorf("trellis: need buffer of at least 18 bytes, got %d", len(bytes))
	}
	dibits, err := ExtractDibits(bits)
	if err != nil {
		return err
	}
	deinterleaved, err := Deinterleave(dibits)
	if err != nil {
		return err
	}
	points, err := ConstellationPoints(deinterleaved)
	if err != nil {
		return err
	}
	tribits, err := ExtractTribits(points)
	if err != nil {
		return err
	}
	binary, err := ExtractBinary(tribits)
	if err != nil {
		return err
	}
	copy(bytes, dmr.BitsToBytes(binary))
	return nil
}

// ExtractDibits extracts dibits from bits.
func ExtractDibits(bits []byte) ([]int8, error) {
	if len(bits) != dmr.InfoBits {
		return nil, fmt.Errorf("trellis: expected %d bits, got %d", dmr.InfoBits, len(bits))
	}
	var dibits = make([]int8, 98)

	for i := 0; i < 196; i += 2 {
		o := i / 2
		switch {
		case bits[i] == 0 && bits[i+1] == 1:
			dibits[o] = 3
			break
		case bits[i] == 0 && bits[i+1] == 0:
			dibits[o] = 1
			break
		case bits[i] == 1 && bits[i+1] == 0:
			dibits[o] = -1
			break
		case bits[i] == 1 && bits[i+1] == 1:
			dibits[o] = -3
			break
		}
	}

	return dibits, nil
}

// Deinterleave the dibits according to DMR AI protocol spec. page 130.
func Deinterleave(dibits []int8) ([]int8, error) {
	if dibits == nil {
		return nil, errors.New("trellis: dibits can't be nil")
	}
	if len(dibits) != 98 {
		return nil, fmt.Errorf("trellis: expected 98 dibits, got %d", len(dibits))
	}

	var deinterleaved = make([]int8, 98)
	for i := 0; i < 98; i++ {
		deinterleaved[interleaveMatrix[i]] = dibits[i]
	}
	return deinterleaved, nil
}

// ConstellationPoints decodes the constellation points according to DMR AI protocol spec. page 129.
func ConstellationPoints(dibits []int8) ([]uint8, error) {
	if dibits == nil {
		return nil, errors.New("trellis: dibits can't be nil")
	}
	if len(dibits) != 98 {
		return nil, fmt.Errorf("trellis: expected 98 dibits, got %d", len(dibits))
	}

	var points = make([]uint8, 49)
	for i := 0; i < 98; i += 2 {
		o := i / 2
		switch {
		case dibits[i] == +1 && dibits[i+1] == -1:
			points[o] = 0
			break
		case dibits[i] == -1 && dibits[i+1] == -1:
			points[o] = 1
			break
		case dibits[i] == +3 && dibits[i+1] == -3:
			points[o] = 2
			break
		case dibits[i] == -3 && dibits[i+1] == -3:
			points[o] = 3
			break
		case dibits[i] == -3 && dibits[i+1] == -1:
			points[o] = 4
			break
		case dibits[i] == +3 && dibits[i+1] == -1:
			points[o] = 5
			break
		case dibits[i] == -1 && dibits[i+1] == -3:
			points[o] = 6
			break
		case dibits[i] == +1 && dibits[i+1] == -3:
			points[o] = 7
			break
		case dibits[i] == -3 && dibits[i+1] == +3:
			points[o] = 8
			break
		case dibits[i] == +3 && dibits[i+1] == +3:
			points[o] = 9
			break
		case dibits[i] == -1 && dibits[i+1] == +1:
			points[o] = 10
			break
		case dibits[i] == +1 && dibits[i+1] == +1:
			points[o] = 11
			break
		case dibits[i] == +1 && dibits[i+1] == +3:
			points[o] = 12
			break
		case dibits[i] == -1 && dibits[i+1] == +3:
			points[o] = 13
			break
		case dibits[i] == +3 && dibits[i+1] == +1:
			points[o] = 14
			break
		case dibits[i] == -3 && dibits[i+1] == +1:
			points[o] = 15
			break
		}
	}

	return points, nil
}

// ExtractTribits maps constellation points to Trellis tribits according to DMR AI protocol spec. page 129.
func ExtractTribits(points []uint8) ([]uint8, error) {
	var (
		match   bool
		last    uint8
		start   int
		tribits = make([]uint8, 48)
	)

	for i := 0; i < 48; i++ {
		start = int(last) * 8
		match = false
		for j := start; j < (start + 8); j++ {
			// Check if this constellation point matches an element of this row of the state table.
			if points[i] == encoderStateTransition[j] {
				match = true
				last = uint8(j - start)
				tribits[i] = uint8(last)
			}
		}

		if !match {
			return nil, fmt.Errorf("trellis: trellis tribit extract error at point %d, data is corrupted", i)
		}
	}

	return tribits, nil
}

// ExtractBinary maps the tribits back to bits.
func ExtractBinary(tribits []uint8) ([]byte, error) {
	if tribits == nil {
		return nil, errors.New("trellis: tribits can't be nill")
	}
	if len(tribits) != 48 {
		return nil, fmt.Errorf("trellis: tribits length is %d, expected 48", len(tribits))
	}

	var bits = make([]byte, 196)
	for i := 0; i < 144; i += 3 {
		o := i / 3

		if tribits[o]&dmr.B00000100 > 0 {
			bits[i] = 1
		}
		if tribits[o]&dmr.B00000010 > 0 {
			bits[i+1] = 1
		}
		if tribits[o]&dmr.B00000001 > 0 {
			bits[i+2] = 1
		}
	}

	return bits, nil
}
