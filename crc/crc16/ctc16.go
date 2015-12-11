// Package crc16 implements the 16-bit cyclic redundancy check, or CRC-16,
// checksum. See http://en.wikipedia.org/wiki/Cyclic_redundancy_check for
// information.
package crc16

// Predefined polynomnials
const (
	// Used by X.25, V.41, HDLC FCS, XMODEM, Bluetooth, PACTOR, SD, ...
	CCITT = 0x8408
)

// Table is a 256-word table representing the polynomial for efficient processing.
type Table [256]uint16

var (
	CCITTTable = makeTable(CCITT)
)

// MakeTable returns the Table constructed from the specified polynomial.
func MakeTable(poly uint16) *Table {
	return makeTable(poly)
}

// makeTable returns the Table constructed from the specified polynomial.
func makeTable(poly uint16) *Table {
	t := new(Table)
	for i := 0; i < 256; i++ {
		crc := uint16(i)
		for j := 0; j < 8; j++ {
			if crc&1 == 1 {
				crc = (crc >> 1) ^ poly
			} else {
				crc >>= 1
			}
		}
		t[i] = crc
	}
	return t
}

// Update returns the result of adding the bytes in p to the crc.
func Update(crc uint16, tab *Table, p []byte) uint16 {
	return update(crc, tab, p)
}

func update(crc uint16, tab *Table, p []byte) uint16 {
	crc = ^crc
	for _, v := range p {
		crc = tab[byte(crc)^v] ^ (crc >> 8)
	}
	return ^crc
}

// Checksum returns the CRC-16 checksum of data using the polynomial represented by the Table.
func Checksum(data []byte, tab *Table) uint16 {
	return Update(0, tab, data)
}

// ChecksumCCITT returns the CRC-16 checksum of data using the CCITT polynomial.
func ChecksumCCITT(data []byte) uint16 {
	return update(0, CCITTTable, data)
}
