package dmr

// Repeater implements a repeater station.
type Repeater interface {
	Active() bool
	Close() error
	ListenAndServe() error

	GetPacketFunc() PacketFunc
	SetPacketFunc(PacketFunc)
}
