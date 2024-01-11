package fec

import (
	"errors"
	"fmt"
)

const (
	RS_12_9_DATASIZE     = 9
	RS_12_9_CHECKSUMSIZE = 3
	// Maximum degree of various polynomials
	RS_12_9_POLY_MAXDEG = RS_12_9_CHECKSUMSIZE * 2
)

type RS_12_9_Poly [RS_12_9_POLY_MAXDEG]uint8

var (
	// DMR AI. spec. page 138.
	rs_12_9_galois_exp_table = [256]uint8{
		0x01, 0x02, 0x04, 0x08, 0x10, 0x20, 0x40, 0x80, 0x1d, 0x3a, 0x74, 0xe8, 0xcd, 0x87, 0x13, 0x26,
		0x4c, 0x98, 0x2d, 0x5a, 0xb4, 0x75, 0xea, 0xc9, 0x8f, 0x03, 0x06, 0x0c, 0x18, 0x30, 0x60, 0xc0,
		0x9d, 0x27, 0x4e, 0x9c, 0x25, 0x4a, 0x94, 0x35, 0x6a, 0xd4, 0xb5, 0x77, 0xee, 0xc1, 0x9f, 0x23,
		0x46, 0x8c, 0x05, 0x0a, 0x14, 0x28, 0x50, 0xa0, 0x5d, 0xba, 0x69, 0xd2, 0xb9, 0x6f, 0xde, 0xa1,
		0x5f, 0xbe, 0x61, 0xc2, 0x99, 0x2f, 0x5e, 0xbc, 0x65, 0xca, 0x89, 0x0f, 0x1e, 0x3c, 0x78, 0xf0,
		0xfd, 0xe7, 0xd3, 0xbb, 0x6b, 0xd6, 0xb1, 0x7f, 0xfe, 0xe1, 0xdf, 0xa3, 0x5b, 0xb6, 0x71, 0xe2,
		0xd9, 0xaf, 0x43, 0x86, 0x11, 0x22, 0x44, 0x88, 0x0d, 0x1a, 0x34, 0x68, 0xd0, 0xbd, 0x67, 0xce,
		0x81, 0x1f, 0x3e, 0x7c, 0xf8, 0xed, 0xc7, 0x93, 0x3b, 0x76, 0xec, 0xc5, 0x97, 0x33, 0x66, 0xcc,
		0x85, 0x17, 0x2e, 0x5c, 0xb8, 0x6d, 0xda, 0xa9, 0x4f, 0x9e, 0x21, 0x42, 0x84, 0x15, 0x2a, 0x54,
		0xa8, 0x4d, 0x9a, 0x29, 0x52, 0xa4, 0x55, 0xaa, 0x49, 0x92, 0x39, 0x72, 0xe4, 0xd5, 0xb7, 0x73,
		0xe6, 0xd1, 0xbf, 0x63, 0xc6, 0x91, 0x3f, 0x7e, 0xfc, 0xe5, 0xd7, 0xb3, 0x7b, 0xf6, 0xf1, 0xff,
		0xe3, 0xdb, 0xab, 0x4b, 0x96, 0x31, 0x62, 0xc4, 0x95, 0x37, 0x6e, 0xdc, 0xa5, 0x57, 0xae, 0x41,
		0x82, 0x19, 0x32, 0x64, 0xc8, 0x8d, 0x07, 0x0e, 0x1c, 0x38, 0x70, 0xe0, 0xdd, 0xa7, 0x53, 0xa6,
		0x51, 0xa2, 0x59, 0xb2, 0x79, 0xf2, 0xf9, 0xef, 0xc3, 0x9b, 0x2b, 0x56, 0xac, 0x45, 0x8a, 0x09,
		0x12, 0x24, 0x48, 0x90, 0x3d, 0x7a, 0xf4, 0xf5, 0xf7, 0xf3, 0xfb, 0xeb, 0xcb, 0x8b, 0x0b, 0x16,
		0x2c, 0x58, 0xb0, 0x7d, 0xfa, 0xe9, 0xcf, 0x83, 0x1b, 0x36, 0x6c, 0xd8, 0xad, 0x47, 0x8e, 0x01,
	}
	// DMR AI. spec. page 138.
	rs_12_9_galois_log_table = [256]uint8{
		0, 0, 1, 25, 2, 50, 26, 198, 3, 223, 51, 238, 27, 104, 199, 75,
		4, 100, 224, 14, 52, 141, 239, 129, 28, 193, 105, 248, 200, 8, 76, 113,
		5, 138, 101, 47, 225, 36, 15, 33, 53, 147, 142, 218, 240, 18, 130, 69,
		29, 181, 194, 125, 106, 39, 249, 185, 201, 154, 9, 120, 77, 228, 114, 166,
		6, 191, 139, 98, 102, 221, 48, 253, 226, 152, 37, 179, 16, 145, 34, 136,
		54, 208, 148, 206, 143, 150, 219, 189, 241, 210, 19, 92, 131, 56, 70, 64,
		30, 66, 182, 163, 195, 72, 126, 110, 107, 58, 40, 84, 250, 133, 186, 61,
		202, 94, 155, 159, 10, 21, 121, 43, 78, 212, 229, 172, 115, 243, 167, 87,
		7, 112, 192, 247, 140, 128, 99, 13, 103, 74, 222, 237, 49, 197, 254, 24,
		227, 165, 153, 119, 38, 184, 180, 124, 17, 68, 146, 217, 35, 32, 137, 46,
		55, 63, 209, 91, 149, 188, 207, 205, 144, 135, 151, 178, 220, 252, 190, 97,
		242, 86, 211, 171, 20, 42, 93, 158, 132, 60, 57, 83, 71, 109, 65, 162,
		31, 45, 67, 216, 183, 123, 164, 118, 196, 23, 73, 236, 127, 12, 111, 246,
		108, 161, 59, 82, 41, 157, 85, 170, 251, 96, 134, 177, 187, 204, 62, 90,
		203, 89, 95, 176, 156, 169, 160, 81, 11, 245, 22, 235, 122, 117, 44, 215,
		79, 174, 213, 233, 230, 231, 173, 232, 116, 214, 244, 234, 168, 80, 88, 175
	}
)

func RS_12_9_Galois_Inv(elt uint8) uint8 {
	return rs_12_9_galois_exp_table[255-rs_12_9_galois_log_table[elt]]
}

func RS_12_9_Galois_Mul(a, b uint8) uint8 {
	if a == 0 || b == 0 {
		return 0
	}
	return rs_12_9_galois_exp_table[(rs_12_9_galois_log_table[a]+rs_12_9_galois_log_table[b])%255]
}

// Multiply by z (shift right by 1).
func RS_12_9_MulPolyZ(poly *RS_12_9_Poly) {
	for i := RS_12_9_POLY_MAXDEG - 1; i > 0; i-- {
		poly[i] = poly[i-1]
	}
	poly[0] = 0
}

func RS_12_9_MulPolys(p1, p2 *RS_12_9_Poly, dst []uint8) {
	var (
		i, j uint8
		tmp  = make([]uint8, RS_12_9_POLY_MAXDEG*2)
	)

	for i = 0; i < RS_12_9_POLY_MAXDEG*2; i++ {
		dst[i] = 0
	}

	for i = 0; i < RS_12_9_POLY_MAXDEG; i++ {
		for j := RS_12_9_POLY_MAXDEG; j < (RS_12_9_POLY_MAXDEG * 2); j++ {
			tmp[j] = 0
		}

		// Scale tmp by p1[i]
		for j = 0; j < RS_12_9_POLY_MAXDEG; j++ {
			tmp[j] = RS_12_9_Galois_Mul(p2[j], p1[i])
		}

		// Shift (multiply) tmp right by i
		for j = (RS_12_9_POLY_MAXDEG * 2) - 1; j >= i && j < (RS_12_9_POLY_MAXDEG*2)-1; j-- {
			tmp[j] = tmp[j-i]
		}
		for j = 0; j < i; j++ {
			tmp[j] = 0
		}

		// Add into partial product
		for j = 0; j < (RS_12_9_POLY_MAXDEG * 2); j++ {
			dst[j] ^= tmp[j]
		}
	}
}

// Computes the combined erasure/error evaluator polynomial (error_locator_poly*syndrome mod z^4)
func RS_12_9_CalcErrorEvaluatorPoly(locator, syndrome, evaluator *RS_12_9_Poly) {
	var (
		i       uint8
		product = make([]uint8, RS_12_9_POLY_MAXDEG*2)
	)

	RS_12_9_MulPolys(locator, syndrome, product)
	for i = 0; i < RS_12_9_CHECKSUMSIZE; i++ {
		evaluator[i] = product[i]
	}
	for ; i < RS_12_9_POLY_MAXDEG; i++ {
		evaluator[i] = 0
	}
}

func RS_12_9_CalcDiscrepancy(locator, syndrome *RS_12_9_Poly, L, n uint8) uint8 {
	var i, sum uint8

	for i = 0; i <= L; i++ {
		sum ^= RS_12_9_Galois_Mul(locator[i], syndrome[n-i])
	}

	return sum
}

// This finds the coefficients of the error locator polynomial, and then calculates
// the error evaluator polynomial using the Berlekamp-Massey algorithm.
// From  Cain, Clark, "Error-Correction Coding For Digital Communications", pp. 216.
func RS_12_9_Calc(syndrome, locator, evaluator *RS_12_9_Poly) {
	var (
		n, L, L2 uint8
		k        int8
		d, i     uint8
		psi2     = make([]uint8, RS_12_9_POLY_MAXDEG)
		D        = RS_12_9_Poly{0, 1, 0}
	)

	k = -1
	for i = 0; i < RS_12_9_POLY_MAXDEG; i++ {
		locator[i] = 0
	}
	locator[0] = 1

	for n = 0; n < RS_12_9_CHECKSUMSIZE; n++ {
		d = RS_12_9_CalcDiscrepancy(locator, syndrome, L, n)
		if d != 0 {
			// psi2 = locator - d*D
			for i = 0; i < RS_12_9_POLY_MAXDEG; i++ {
				psi2[i] = locator[i] ^ RS_12_9_Galois_Mul(d, D[i])
			}

			if int8(L) < int8(n)-k {
				L2 = uint8(int8(n) - k)
				k = int8(int8(n) - int8(L))
				for i = 0; i < RS_12_9_POLY_MAXDEG; i++ {
					D[i] = RS_12_9_Galois_Mul(locator[i], RS_12_9_Galois_Inv(d))
				}
				L = L2
			}

			// locator = psi2
			for i = 0; i < RS_12_9_POLY_MAXDEG; i++ {
				locator[i] = psi2[i]
			}
		}
		RS_12_9_MulPolyZ(&D)
	}
	RS_12_9_CalcErrorEvaluatorPoly(locator, syndrome, evaluator)
}

// The error-locator polynomial's roots are found by looking for the values of a^n where
// evaluating the polynomial yields zero (evaluating rs_12_9_error_locator_poly at
// successive values of alpha (Chien's search)).
func RS_12_9_FindRoots(locator *RS_12_9_Poly) []uint8 {
	var k, r uint16
	roots := make([]uint8, 0)
	for r = 1; r < 256; r++ {
		var sum uint8
		// Evaluate locator at r
		for k = 0; k < RS_12_9_CHECKSUMSIZE+1; k++ {
			sum ^= RS_12_9_Galois_Mul(rs_12_9_galois_exp_table[(k*r)%255], locator[k])
		}

		if sum == 0 {
			roots = append(roots, uint8(255-r))
		}
	}

	return roots
}

func RS_12_9_CalcSyndrome(data []byte, syndrome *RS_12_9_Poly) error {
	if len(data) != RS_12_9_DATASIZE+RS_12_9_CHECKSUMSIZE {
		return fmt.Errorf("fec/rs_12_9: unexpected size %d, expected %d bytes",
			len(data), RS_12_9_DATASIZE+RS_12_9_CHECKSUMSIZE)
	}

	var i, j uint8
	for i = 0; i < 3; i++ {
		syndrome[i] = 0
	}

	for j = 0; j < 3; j++ {
		for i = 0; i < uint8(len(data)); i++ {
			syndrome[j] = data[i] ^ RS_12_9_Galois_Mul(rs_12_9_galois_exp_table[j+1], syndrome[j])
		}
	}

	return nil
}

func RS_12_9_CheckSyndrome(syndrome *RS_12_9_Poly) bool {
	for _, v := range syndrome {
		if v != 0 {
			return true
		}
	}
	return false
}

func RS_12_9_Correct(data []byte, syndrome *RS_12_9_Poly) (int, error) {
	if len(data) != RS_12_9_DATASIZE+RS_12_9_CHECKSUMSIZE {
		return -1, fmt.Errorf("fec/rs_12_9: unexpected size %d, expected %d bytes",
			len(data), RS_12_9_DATASIZE+RS_12_9_CHECKSUMSIZE)
	}

	var (
		i, j        uint8
		errorsFound int
		locator     = RS_12_9_Poly{}
		evaluator   = RS_12_9_Poly{}
	)
	RS_12_9_Calc(syndrome, &locator, &evaluator)
	roots := RS_12_9_FindRoots(&locator)
	errorsFound = len(roots)

	if errorsFound == 0 {
		return 0, nil
	}

	// Error correction is done using the error-evaluator equation on pp 207.
	if errorsFound > 0 && errorsFound < RS_12_9_CHECKSUMSIZE {
		// First check for illegal error locations.
		for r := 0; r < errorsFound; r++ {
			if roots[r] >= RS_12_9_DATASIZE+RS_12_9_CHECKSUMSIZE {
				return errorsFound, errors.New("fec/rs_12_9: errors can't be corrected")
			}
		}

		// Evaluates rs_12_9_error_evaluator_poly/rs_12_9_error_locator_poly' at the roots
		// alpha^(-i) for error locs i.
		for r := 0; r < errorsFound; r++ {
			i = roots[r]

			var num, denom uint8
			// Evaluate rs_12_9_error_evaluator_poly at alpha^(-i)
			for j = 0; j < RS_12_9_POLY_MAXDEG; j++ {
				num ^= RS_12_9_Galois_Mul(evaluator[j], rs_12_9_galois_exp_table[((255-i)*j)%255])
			}

			// Evaluate rs_12_9_error_evaluator_poly' (derivative) at alpha^(-i). All odd powers disappear.
			for j = 1; j < RS_12_9_POLY_MAXDEG; j += 2 {
				denom ^= RS_12_9_Galois_Mul(locator[j], rs_12_9_galois_exp_table[((255-i)*(j-1))%255])
			}

			data[len(data)-int(i)-1] ^= RS_12_9_Galois_Mul(num, RS_12_9_Galois_Inv(denom))
		}

		return errorsFound, nil
	}

	return 0, nil
}

// Simulates an LFSR with the generator polynomial and calculates checksum bytes for the given data.
func RS_12_9_CalcChecksum(data []byte) []uint8 {
	var (
		feedback uint8
		genpoly  = []uint8{0x40, 0x38, 0x0e, 0x01} // See DMR AI. spec. page 136 for these coefficients.
		checksum = make([]uint8, 3)
	)

	for i := 0; i < 9; i++ {
		feedback = data[i] ^ checksum[0]
		checksum[0] = checksum[1] ^ RS_12_9_Galois_Mul(genpoly[2], feedback)
		checksum[1] = checksum[2] ^ RS_12_9_Galois_Mul(genpoly[1], feedback)
		checksum[2] = RS_12_9_Galois_Mul(genpoly[0], feedback)
	}
	return checksum
}
