package dmr

// Repeater implements a repeater station.
type Repeater interface {
	Active() bool
	Close() error
	ListenAndServe() error
	Send(*Packet) error

	GetPacketFunc() PacketFunc
	SetPacketFunc(PacketFunc)
}
