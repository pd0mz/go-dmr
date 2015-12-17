package dmr

type DataBlock struct {
	Serial uint8
	CRC    uint16
	OK     bool
	Data   [24]byte
	Length uint8
}
