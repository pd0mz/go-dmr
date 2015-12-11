package fec

var (
	hamming15_11_3_gen = [11]uint32{
		0x4009, 0x200d, 0x100f, 0x080e, 0x0407, 0x020a, 0x0105, 0x008b, 0x004c, 0x0026, 0x0013,
	}
	hamming15_11_3_table = [16]uint32{
		0x0000, 0x0001, 0x0002, 0x0013, 0x0004, 0x0105, 0x0026, 0x0407,
		0x0008, 0x4009, 0x020a, 0x008b, 0x004c, 0x200d, 0x080e, 0x100f,
	}
)

// Correct a block of data using Hamming(15, 11, 3).
func Hamming15_11_3_Correct(block *uint32) {
	var (
		ecc, syndrome uint32
		codeword      = *block
	)

	for i := 0; i < 11; i++ {
		if (codeword & hamming15_11_3_gen[i]) > 0x0f {
			ecc ^= hamming15_11_3_gen[i]
		}
	}
	syndrome = ecc ^ codeword

	if syndrome != 0 {
		codeword ^= hamming15_11_3_table[syndrome&0x0f]
	}

	*block = (codeword >> 4)
}

// Hamming(15, 11, 3) encode a block of data.
func Hamming15_11_3_Encode(input uint32) uint32 {
	var codeword uint32
	for i := uint8(0); i < 11; i++ {
		if input&(1<<(10-i)) > 0 {
			codeword ^= hamming15_11_3_gen[i]
		}
	}
	return codeword
}
