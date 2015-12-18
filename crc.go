package dmr

const (
	// G(x) = x^9+x^6+x^4+x^3+1
	crc9poly = 0x59
	// G(x) = x^16+x^12+x^5+1
	crc16poly = 0x1021
	crc32poly = 0x04c11db7
)

func crc9(crc *uint16, b uint8, bits int) {
	var v uint8 = 0x80
	for i := 0; i < 8-bits; i++ {
		v >>= 1
	}
	for i := 0; i < 8; i++ {
		xor := (*crc)&0x0100 > 0
		(*crc) <<= 1
		// Limit the number of shift registers to 9.
		*crc &= 0x01ff
		if b&v > 0 {
			(*crc)++
		}
		if xor {
			(*crc) ^= crc9poly
		}
		v >>= 1
	}
}

func crc9end(crc *uint16, bits int) {
	for i := 0; i < bits; i++ {
		xor := (*crc)&0x100 > 0
		(*crc) <<= 1
		// Limit the number of shift registers to 9.
		*crc &= 0x01ff
		if xor {
			(*crc) ^= crc9poly
		}
	}
}

func crc16(crc *uint16, b byte) {
	var v uint8 = 0x80
	for i := 0; i < 8; i++ {
		xor := ((*crc) & 0x8000) > 0
		(*crc) <<= 1
		if b&v > 0 {
			(*crc)++
		}
		if xor {
			(*crc) ^= crc16poly
		}
		v >>= 1
	}
}

func crc16end(crc *uint16) {
	for i := 0; i < 16; i++ {
		xor := ((*crc) & 0x8000) > 0
		(*crc) <<= 1
		if xor {
			(*crc) ^= crc16poly
		}
	}
}

func crc32(crc *uint32, b byte) {
	var v uint8 = 0x80
	for i := 0; i < 8; i++ {
		xor := ((*crc) & 0x80000000) > 0
		(*crc) <<= 1
		if b&v > 0 {
			(*crc)++
		}
		if xor {
			(*crc) ^= crc32poly
		}
		v >>= 1
	}
}

func crc32end(crc *uint32) {
	for i := 0; i < 32; i++ {
		xor := ((*crc) & 0x80000000) > 0
		(*crc) <<= 1
		if xor {
			(*crc) ^= crc32poly
		}
	}
}
