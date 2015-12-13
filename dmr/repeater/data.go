package repeater

// DataFrame is a decoded frame with data
type DataFrame struct {
	SrcID, DstID uint32
	Timeslot     uint8
	Data         []byte
}
