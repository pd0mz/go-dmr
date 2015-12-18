package dmr

import "testing"

func TestCRC9(t *testing.T) {
	tests := map[uint16][]byte{
		0x0000: []byte{},
		0x0100: []byte{0x00, 0x01},
		0x0179: []byte("hello world"),
	}

	for want, test := range tests {
		var crc uint16
		for _, b := range test {
			crc9(&crc, b, 8)
		}
		crc9end(&crc, 8)
		if crc != want {
			t.Fatalf("crc9 %v failed: %#04x != %#04x", test, crc, want)
		}
	}
}

func TestCRC16(t *testing.T) {
	tests := map[uint16][]byte{
		0x0000: []byte{},
		0x1021: []byte{0x00, 0x01},
		0x3be4: []byte("hello world"),
	}

	for want, test := range tests {
		var crc uint16
		for _, b := range test {
			crc16(&crc, b)
		}
		crc16end(&crc)
		if crc != want {
			t.Fatalf("crc16 %v failed: %#04x != %#04x", test, crc, want)
		}
	}
}

func TestCRC32(t *testing.T) {
	tests := map[uint32][]byte{
		0x00000000: []byte{},
		0x04c11db7: []byte{0x00, 0x01},
		0x737af2ae: []byte("hello world"),
	}

	for want, test := range tests {
		var crc uint32
		for _, b := range test {
			crc32(&crc, b)
		}
		crc32end(&crc)
		if crc != want {
			t.Fatalf("crc32 %v failed: %#08x != %#08x", test, crc, want)
		}
	}
}
