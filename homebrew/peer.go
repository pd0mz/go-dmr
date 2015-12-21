package homebrew

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"time"

	"github.com/pd0mz/go-dmr"
)

// Peer is a remote repeater that also speaks the Homebrew protocol
type Peer struct {
	ID                  uint32
	Addr                *net.UDPAddr
	AuthKey             []byte
	Status              AuthStatus
	Nonce               []byte
	Token               []byte
	Incoming            bool
	UnlinkOnAuthFailure bool
	PacketReceived      dmr.PacketFunc
	Last                struct {
		PacketSent     time.Time
		PacketReceived time.Time
		PingSent       time.Time
		PingReceived   time.Time
		PongReceived   time.Time
	}

	// Packed repeater ID
	id []byte
}

func (p *Peer) CheckRepeaterID(id []byte) bool {
	return id != nil && p.id != nil && bytes.Equal(id, p.id)
}

func (p *Peer) UpdateToken(nonce []byte) {
	p.Nonce = nonce
	hash := sha256.New()
	hash.Write(p.Nonce)
	hash.Write(p.AuthKey)
	p.Token = []byte(hex.EncodeToString(hash.Sum(nil)))
}
