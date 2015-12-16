package homebrew

type Config struct {
	// ID is the local DMR ID.
	ID uint32

	// PeerID is the remote DMR ID.
	PeerID uint32

	// AuthKey is the shared secret.
	AuthKey string
}
