// Package homebrew implements the Home Brew DMR IPSC protocol
package homebrew

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/tehmaze/go-dmr"
)

var logger *log.Logger

type AuthStatus uint8

const (
	AuthNone AuthStatus = iota
	AuthBegin
	AuthDone
	AuthFailed
)

// Messages as documented by DL5DI, G4KLX and DG1HT, see "DMRplus IPSC Protocol for HB repeater (20150726).pdf".
var (
	DMRData         = []byte("DMRD")
	MasterNAK       = []byte("MSTNAK")
	MasterACK       = []byte("MSTACK")
	RepeaterLogin   = []byte("RPTL")
	RepeaterKey     = []byte("RPTK")
	MasterPing      = []byte("MSTPING")
	RepeaterPong    = []byte("RPTPONG")
	MasterClosing   = []byte("MSTCL")
	RepeaterClosing = []byte("RPTCL")
)

// We ping the peers every minute
var (
	AuthTimeout                   = time.Second * 90
	PingInterval                  = time.Minute
	PingTimeout                   = time.Second * 150
	RepeaterConfigurationInterval = time.Minute * 5
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
		PacketSent                time.Time
		PacketReceived            time.Time
		PingSent                  time.Time
		PongReceived              time.Time
		RepeaterConfigurationSent time.Time
	}
}

func (p *Peer) UpdateToken(nonce []byte) {
	p.Nonce = nonce
	hash := sha256.New()
	hash.Write(p.Nonce)
	hash.Write(p.AuthKey)
	p.Token = []byte(hex.EncodeToString(hash.Sum(nil)))
}

// Homebrew is implements the Homebrew IPSC DMR Air Interface protocol
type Homebrew struct {
	Config         *RepeaterConfiguration
	Peer           map[string]*Peer
	PeerID         map[uint32]*Peer
	PacketReceived dmr.PacketFunc

	conn   *net.UDPConn
	closed bool
	id     []byte
	mutex  *sync.Mutex
	stop   chan bool
}

// New creates a new Homebrew repeater
func New(config *RepeaterConfiguration, addr *net.UDPAddr) (*Homebrew, error) {
	var err error

	if config == nil {
		return nil, errors.New("homebrew: RepeaterConfiguration can't be nil")
	}

	h := &Homebrew{
		Config: config,
		Peer:   make(map[string]*Peer),
		PeerID: make(map[uint32]*Peer),
		id:     []byte(fmt.Sprintf("%08x", config.ID)),
		mutex:  &sync.Mutex{},
	}
	if h.conn, err = net.ListenUDP("udp", addr); err != nil {
		return nil, errors.New("homebrew: " + err.Error())
	}

	return h, nil
}

func (h *Homebrew) Active() bool {
	return !h.closed && h.conn != nil
}

// Close stops the active listeners
func (h *Homebrew) Close() error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if !h.Active() {
		return nil
	}

	// Kill keepalive goroutine
	if h.stop != nil {
		close(h.stop)
		h.stop = nil
	}

	// Kill listening socket
	h.closed = true
	return h.conn.Close()
}

// Link establishes a new link with a peer
func (h *Homebrew) Link(peer *Peer) error {
	if peer.Addr == nil {
		return errors.New("homebrew: peer Addr can't be nil")
	}
	if peer.AuthKey == nil || len(peer.AuthKey) == 0 {
		return errors.New("homebrew: peer AuthKey can't be nil")
	}

	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Reset state
	peer.Last.PacketSent = time.Time{}
	peer.Last.PacketReceived = time.Time{}
	peer.Last.PingSent = time.Time{}
	peer.Last.PongReceived = time.Time{}

	// Register our peer
	h.Peer[peer.Addr.String()] = peer
	h.PeerID[peer.ID] = peer

	return h.handleAuth(peer)
}

func (h *Homebrew) Unlink(id uint32) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	peer, ok := h.PeerID[id]
	if !ok {
		return fmt.Errorf("homebrew: peer %d not linked", id)
	}

	delete(h.Peer, peer.Addr.String())
	delete(h.PeerID, id)
	return nil
}

func (h *Homebrew) ListenAndServe() error {
	var data = make([]byte, 53)

	h.stop = make(chan bool)
	go h.keepalive(h.stop)

	h.closed = false
	for !h.closed {
		n, peer, err := h.conn.ReadFromUDP(data)
		if err != nil {
			return err
		}
		if err := h.handle(peer, data[:n]); err != nil {
			return err
		}
	}

	return nil
}

func (h *Homebrew) WritePacketToPeer(p *dmr.Packet, peer *Peer) error {
	return h.WriteToPeer(h.parsePacket(p), peer)
}

func (h *Homebrew) WriteToPeer(b []byte, peer *Peer) error {
	if peer == nil {
		return errors.New("homebrew: can't write to nil peer")
	}

	peer.Last.PacketSent = time.Now()
	_, err := h.conn.WriteTo(b, peer.Addr)
	return err
}

func (h *Homebrew) WriteToPeerWithID(b []byte, id uint32) error {
	return h.WriteToPeer(b, h.getPeer(id))
}

func (h *Homebrew) getPeer(id uint32) *Peer {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if peer, ok := h.PeerID[id]; ok {
		return peer
	}

	return nil
}

func (h *Homebrew) getPeerByAddr(addr *net.UDPAddr) *Peer {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if peer, ok := h.Peer[addr.String()]; ok {
		return peer
	}

	return nil
}

func (h *Homebrew) getPeers() []*Peer {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	var peers = make([]*Peer, 0)
	for _, peer := range h.Peer {
		peers = append(peers, peer)
	}

	return peers
}

func (h *Homebrew) handle(remote *net.UDPAddr, data []byte) error {
	peer := h.getPeerByAddr(remote)
	if peer == nil {
		logger.Printf("ignored packet from unknown peer %s\n", remote)
		return nil
	}

	// Ignore packet that are clearly invalid, this is the minimum packet length for any Homebrew protocol frame
	if len(data) < 14 {
		return nil
	}

	if peer.Status != AuthDone {
		// Ignore DMR data at this stage
		if bytes.Equal(data[:4], DMRData) {
			return nil
		}

		if peer.Incoming {
			// Verify we have a matching peer ID
			id, err := h.parseRepeaterID(data[4:])
			if err != nil {
				logger.Printf("peer %d@%s sent invalid repeater ID (ignored)\n", peer.ID, remote)
				return nil
			}
			var ok = id == peer.ID
			if !ok {
				logger.Printf("peer %d@%s sent unexpected repeater ID %d\n", peer.ID, remote, id)
			}

			switch peer.Status {
			case AuthNone:
				switch {
				case bytes.Equal(data[:4], RepeaterLogin):
					if !ok {
						return h.WriteToPeer(append(MasterNAK, h.id...), peer)
					}

					// Peer is verified, generate a nonce
					nonce := make([]byte, 4)
					if _, err := rand.Read(nonce); err != nil {
						logger.Printf("peer %d@%s nonce generation failed: %v\n", peer.ID, remote, err)
						return h.WriteToPeer(append(MasterNAK, h.id...), peer)
					}

					peer.UpdateToken(nonce)
					peer.Status = AuthBegin
					return h.WriteToPeer(append(append(MasterACK, h.id...), nonce...), peer)

				default:
					// Ignore unauthenticated repeater, we're not going to reply unless it's
					// an actual login request; if it was indeed a valid repeater and we missed
					// anything, we rely on the remote end to retry to reconnect if it doesn't
					// get an answer in a timely manner.
					break
				}
				break

			case AuthBegin:
				switch {
				case bytes.Equal(data[:4], RepeaterKey):
					if ok && len(data) != 76 {
						logger.Printf("peer %d@%s sent invalid key challenge length of %d\n", peer.ID, remote, len(data))
						ok = false
					}
					if !ok {
						peer.Status = AuthNone
						return h.WriteToPeer(append(MasterNAK, h.id...), peer)
					}

					if !bytes.Equal(data[12:], peer.Token) {
						logger.Printf("peer %d@%s sent invalid key challenge token\n", peer.ID, remote)
						peer.Status = AuthNone
						return h.WriteToPeer(append(MasterNAK, h.id...), peer)
					}

					peer.Status = AuthDone
					return h.WriteToPeer(append(MasterACK, h.id...), peer)
				}
			}
		} else {
			// Verify we have a matching peer ID
			id, err := h.parseRepeaterID(data[6:14])
			if err != nil {
				logger.Printf("peer %d@%s sent invalid repeater ID (ignored)\n", peer.ID, remote)
				return nil
			}
			if id != peer.ID {
				logger.Printf("peer %d@%s sent unexpected repeater ID %d (ignored)\n", peer.ID, remote, id)
				return nil
			}

			switch peer.Status {
			case AuthNone:
				switch {
				case bytes.Equal(data[:6], MasterACK):
					logger.Printf("peer %d@%s sent nonce\n", peer.ID, remote)
					peer.Status = AuthBegin
					peer.UpdateToken(data[14:])
					return h.handleAuth(peer)

				case bytes.Equal(data[:6], MasterNAK):
					logger.Printf("peer %d@%s refused login\n", peer.ID, remote)
					peer.Status = AuthFailed
					if peer.UnlinkOnAuthFailure {
						h.Unlink(peer.ID)
					}
					break

				default:
					logger.Printf("peer %d@%s sent unexpected login reply (ignored)\n", peer.ID, remote)
					break
				}

			case AuthBegin:
				switch {
				case bytes.Equal(data[:6], MasterACK):
					logger.Printf("peer %d@%s accepted login\n", peer.ID, remote)
					peer.Status = AuthDone
					peer.Last.RepeaterConfigurationSent = time.Now()
					return h.WriteToPeer(h.Config.Bytes(), peer)

				case bytes.Equal(data[:6], MasterNAK):
					logger.Printf("peer %d@%s refused login\n", peer.ID, remote)
					peer.Status = AuthFailed
					if peer.UnlinkOnAuthFailure {
						h.Unlink(peer.ID)
					}
					break

				default:
					logger.Printf("peer %d@%s sent unexpected login reply (ignored)\n", peer.ID, remote)
					break
				}
			}
		}
	} else {
		// Authentication is done
		switch {
		case bytes.Equal(data[:4], DMRData):
			p, err := h.parseData(data[4:])
			if err != nil {
				return err
			}
			return h.handlePacket(p, peer)

		case peer.Incoming && len(data) == 15 && bytes.Equal(data[:7], MasterPing):
			// Verify we have a matching peer ID
			id, err := h.parseRepeaterID(data[7:])
			if err != nil {
				logger.Printf("peer %d@%s sent invalid repeater ID (ignored)\n", peer.ID, remote)
				return nil
			}
			if id != peer.ID {
				logger.Printf("peer %d@%s sent unexpected repeater ID %d (ignored)\n", peer.ID, remote, id)
				return nil
			}
			return h.WriteToPeer(append(RepeaterPong, h.id...), peer)

		default:
			logger.Printf("peer %d@%s sent unexpected packet\n", peer.ID, remote)
			break
		}
	}

	return nil
}

func (h *Homebrew) handleAuth(peer *Peer) error {
	if !peer.Incoming {
		switch peer.Status {
		case AuthNone:
			// Send login packet
			return h.WriteToPeer(append(RepeaterLogin, h.id...), peer)

		case AuthBegin:
			// Send repeater key exchange packet
			return h.WriteToPeer(append(append(RepeaterKey, h.id...), peer.Token...), peer)
		}
	}
	return nil
}

func (h *Homebrew) handlePacket(p *dmr.Packet, peer *Peer) error {
	if peer.PacketReceived != nil {
		return peer.PacketReceived(h, p)
	}
	if h.PacketReceived == nil {
		return errors.New("homebrew: no PacketReceived func defined to handle DMR packet")
	}
	return h.PacketReceived(h, p)
}

func (h *Homebrew) keepalive(stop <-chan bool) {
	for {
		select {
		case <-time.After(time.Second):
			now := time.Now()

			for _, peer := range h.getPeers() {
				// Ping protocol only applies to outgoing links, and also the auth retries
				// are entirely up to the peer.
				if peer.Incoming {
					continue
				}

				switch peer.Status {
				case AuthNone, AuthBegin:
					switch {
					case now.Sub(peer.Last.PacketReceived) > AuthTimeout:
						logger.Printf("peer %d@%s not responding to login; retrying\n", peer.ID, peer.Addr)
						if err := h.handleAuth(peer); err != nil {
							logger.Printf("peer %d@%s retry failed: %v\n", peer.ID, peer.Addr, err)
						}
						break
					}

				case AuthDone:
					switch {
					case now.Sub(peer.Last.PongReceived) > PingTimeout:
						peer.Status = AuthNone
						logger.Printf("peer %d@%s not responding to ping; trying to re-establish connection", peer.ID, peer.Addr)
						if err := h.handleAuth(peer); err != nil {
							logger.Printf("peer %d@%s retry failed: %v\n", peer.ID, peer.Addr, err)
						}
						break

					case now.Sub(peer.Last.PingSent) > PingInterval:
						peer.Last.PingSent = now
						if err := h.WriteToPeer(append(MasterPing, h.id...), peer); err != nil {
							logger.Printf("peer %d@%s ping failed: %v\n", peer.ID, peer.Addr, err)
						}
						break

					case now.Sub(peer.Last.RepeaterConfigurationSent) > RepeaterConfigurationInterval:
						peer.Last.RepeaterConfigurationSent = time.Now()
						if err := h.WriteToPeer(h.Config.Bytes(), peer); err != nil {
							logger.Printf("peer %d@%s repeater configuration failed: %v\n", peer.ID, peer.Addr, err)
						}
						break
					}
				}
			}

		case <-stop:
			return
		}
	}
}

// parseData converts Homebrew packet format to DMR packet format
func (h *Homebrew) parseData(data []byte) (*dmr.Packet, error) {
	if len(data) != 53 {
		return nil, fmt.Errorf("homebrew: expected 53 data bytes, got %d", len(data))
	}

	var p = &dmr.Packet{
		Sequence: data[4],
		SrcID:    uint32(data[5])<<16 | uint32(data[6])<<8 | uint32(data[7]),
		DstID:    uint32(data[8])<<16 | uint32(data[9])<<8 | uint32(data[10]),
	}
	p.SetData(data[20:])

	return p, nil
}

// parsePacket converts DMR packet format to Homebrew packet format suitable for sending on the wire
func (h *Homebrew) parsePacket(p *dmr.Packet) []byte {
	var d = make([]byte, 53)

	// Signature, 4 bytes, "DMRD"
	copy(d[0:], DMRData)

	// Seq No, 1 byte
	d[4] = p.Sequence

	// Src ID, 3 bytes
	d[5] = uint8((p.SrcID >> 16) & 0xff)
	d[6] = uint8((p.SrcID >> 8) & 0xff)
	d[7] = uint8((p.SrcID & 0xff))

	// Dst ID, 3 bytes
	d[8] = uint8((p.DstID >> 16) & 0xff)
	d[9] = uint8((p.DstID >> 8) & 0xff)
	d[10] = uint8((p.DstID & 0xff))

	// RptrID, 4 bytes
	binary.LittleEndian.PutUint32(d[11:], p.RepeaterID)

	var s byte
	s |= (p.Timeslot & 0x01)      // Slot no, 1 bit
	s |= (p.CallType & 0x01) << 1 // Call Type, 1 bit
	switch p.DataType {           // Frame Type, 2 bits and Data Type or Voice Seq, 4 bits
	case dmr.VoiceBurstA:
		s |= 0x01 << 2 // Voice sync
		s |= (p.DataType - dmr.VoiceBurstA) << 4
	case dmr.VoiceBurstB, dmr.VoiceBurstC, dmr.VoiceBurstD, dmr.VoiceBurstE, dmr.VoiceBurstF:
		s |= 0x00 << 2 // Voice (no-op)
		s |= (p.DataType - dmr.VoiceBurstA) << 4
	default:
		s |= 0x02 << 2 // Data
		s |= p.DataType << 4
	}

	// StreamID, 4 bytes
	binary.BigEndian.PutUint32(d[16:], p.StreamID)

	// DMR Data, 33 bytes
	copy(d[20:], p.Data)
	return d
}

func (h *Homebrew) parseRepeaterID(data []byte) (uint32, error) {
	id, err := strconv.ParseUint(string(data), 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(id), nil
}

// Interface compliance check
var _ dmr.Repeater = (*Homebrew)(nil)

// UpdateLogger replaces the package logger.
func UpdateLogger(l *log.Logger) {
	logger = l
}

func init() {
	UpdateLogger(log.New(os.Stderr, "dmr/homebrew: ", log.LstdFlags))
}
