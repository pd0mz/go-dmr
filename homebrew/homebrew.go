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
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/op/go-logging"
	"github.com/pd0mz/go-dmr"
)

var log = logging.MustGetLogger("dmr/homebrew")

type AuthStatus uint8

func (a *AuthStatus) String() string {
	switch *a {
	case AuthNone:
		return "none"
	case AuthBegin:
		return "begin"
	case AuthDone:
		return "none"
	case AuthFailed:
		return "failed"
	default:
		return "invalid"
	}
}

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
	AuthTimeout  = time.Second * 5
	PingInterval = time.Second * 5
	PingTimeout  = time.Second * 15
	SendInterval = time.Millisecond * 30
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

// Homebrew is implements the Homebrew IPSC DMR Air Interface protocol
type Homebrew struct {
	Config *RepeaterConfiguration
	Peer   map[string]*Peer
	PeerID map[uint32]*Peer

	pf     dmr.PacketFunc
	conn   *net.UDPConn
	closed bool
	id     []byte
	last   time.Time   // Record last received frame time
	mutex  *sync.Mutex // Mutex for manipulating peer list or send queue
	rxtx   *sync.Mutex // Mutex for when receiving data or sending data
	stop   chan bool
	queue  []*dmr.Packet
}

// New creates a new Homebrew repeater
func New(config *RepeaterConfiguration, addr *net.UDPAddr) (*Homebrew, error) {
	var err error

	if config == nil {
		return nil, errors.New("homebrew: RepeaterConfiguration can't be nil")
	}
	if addr == nil {
		return nil, errors.New("homebrew: addr can't be nil")
	}

	h := &Homebrew{
		Config: config,
		Peer:   make(map[string]*Peer),
		PeerID: make(map[uint32]*Peer),
		id:     packRepeaterID(config.ID),
		mutex:  &sync.Mutex{},
		queue:  make([]*dmr.Packet, 0),
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

	log.Info("closing")

	// Tell peers we're closing
closing:
	for _, peer := range h.Peer {
		if peer.Status == AuthDone {
			if err := h.WriteToPeer(append(RepeaterClosing, h.id...), peer); err != nil {
				break closing
			}
		}
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
	if peer == nil {
		return errors.New("homebrew: peer can't be nil")
	}
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
	peer.id = packRepeaterID(peer.ID)
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
			if h.closed && strings.HasSuffix(err.Error(), "use of closed network connection") {
				break
			}
			return err
		}
	}

	log.Info("listener closed")
	return nil
}

// Send a packet to the peers. Will block until the packet is sent.
func (h *Homebrew) Send(p *dmr.Packet) error {
	h.rxtx.Lock()
	defer h.rxtx.Unlock()

	data := BuildData(p, h.Config.ID)
	for _, peer := range h.getPeers() {
		if err := h.WriteToPeer(data, peer); err != nil {
			return err
		}
	}

	return nil
}

func (h *Homebrew) GetPacketFunc() dmr.PacketFunc {
	return h.pf
}

func (h *Homebrew) SetPacketFunc(f dmr.PacketFunc) {
	h.pf = f
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

func (h *Homebrew) checkRepeaterID(id []byte) bool {
	return id != nil && bytes.Equal(id, h.id)
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
		log.Debugf("ignored packet from unknown peer %s\n", remote)
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
			switch peer.Status {
			case AuthNone:
				switch {
				case bytes.Equal(data[:4], RepeaterLogin):
					if !peer.CheckRepeaterID(data[4:]) {
						log.Warningf("peer %d@%s sent invalid repeater ID %q (ignored)\n", peer.ID, remote, string(data[4:]))
						return h.WriteToPeer(append(MasterNAK, h.id...), peer)
					}

					// Peer is verified, generate a nonce
					nonce := make([]byte, 4)
					if _, err := rand.Read(nonce); err != nil {
						log.Errorf("peer %d@%s nonce generation failed: %v\n", peer.ID, remote, err)
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
					if !peer.CheckRepeaterID(data[4:]) {
						log.Warningf("peer %d@%s sent invalid repeater ID %q (ignored)\n", peer.ID, remote, string(data[4:]))
						return h.WriteToPeer(append(MasterNAK, h.id...), peer)
					}
					if len(data) != 76 {
						peer.Status = AuthNone
						return h.WriteToPeer(append(MasterNAK, h.id...), peer)
					}
					if !bytes.Equal(data[12:], peer.Token) {
						log.Errorf("peer %d@%s sent invalid key challenge token\n", peer.ID, remote)
						peer.Status = AuthNone
						return h.WriteToPeer(append(MasterNAK, h.id...), peer)
					}

					peer.Last.PingSent = time.Now()
					peer.Last.PongReceived = time.Now()
					peer.Status = AuthDone
					return h.WriteToPeer(append(MasterACK, h.id...), peer)
				}
			}
		} else {
			// Verify we have a matching peer ID
			if !h.checkRepeaterID(data[6:14]) {
				log.Warningf("peer %d@%s sent invalid repeater ID %q (ignored)\n", peer.ID, remote, string(data[6:14]))
				return nil
			}

			switch peer.Status {
			case AuthNone:
				switch {
				case bytes.Equal(data[:6], MasterACK):
					log.Debugf("peer %d@%s sent nonce\n", peer.ID, remote)
					peer.Status = AuthBegin
					peer.UpdateToken(data[14:])
					return h.handleAuth(peer)

				case bytes.Equal(data[:6], MasterNAK):
					log.Errorf("peer %d@%s refused login\n", peer.ID, remote)
					peer.Status = AuthFailed
					if peer.UnlinkOnAuthFailure {
						h.Unlink(peer.ID)
					}
					break

				default:
					log.Warningf("peer %d@%s sent unexpected login reply (ignored)\n", peer.ID, remote)
					break
				}

			case AuthBegin:
				switch {
				case bytes.Equal(data[:6], MasterACK):
					log.Infof("peer %d@%s accepted login\n", peer.ID, remote)
					peer.Status = AuthDone
					peer.Last.PingSent = time.Now()
					peer.Last.PongReceived = time.Now()
					return h.WriteToPeer(h.Config.Bytes(), peer)

				case bytes.Equal(data[:6], MasterNAK):
					log.Errorf("peer %d@%s refused login\n", peer.ID, remote)
					peer.Status = AuthFailed
					if peer.UnlinkOnAuthFailure {
						h.Unlink(peer.ID)
					}
					break

				default:
					log.Warningf("peer %d@%s sent unexpected login reply (ignored)\n", peer.ID, remote)
					break
				}
			}
		}
	} else {
		// Authentication is done
		if peer.Incoming {
			switch {
			case bytes.Equal(data[:4], DMRData):
				p, err := h.parseData(data)
				if err != nil {
					return err
				}
				return h.handlePacket(p, peer)

			case bytes.Equal(data[:6], MasterACK):
				break

			case len(data) == 15 && bytes.Equal(data[:7], MasterPing):
				return h.WriteToPeer(append(RepeaterPong, data[7:]...), peer)

			default:
				log.Warningf("peer %d@%s sent unexpected packet (status=%s):\n", peer.ID, remote, peer.Status.String())
				log.Debug(hex.Dump(data))
				break
			}
		} else {
			switch {
			case bytes.Equal(data[:4], DMRData):
				p, err := h.parseData(data)
				if err != nil {
					return err
				}
				return h.handlePacket(p, peer)

			case bytes.Equal(data[:6], MasterACK):
				if !h.checkRepeaterID(data[6:]) {
					log.Warningf("peer %d@%s sent invalid repeater ID %q (ignored)\n", peer.ID, remote, string(data[6:14]))
					return nil
				}
				peer.Last.PingSent = time.Now()
				return h.WriteToPeer(append(MasterPing, h.id...), peer)

			case bytes.Equal(data[:6], MasterNAK):
				if !h.checkRepeaterID(data[6:]) {
					log.Warningf("peer %d@%s sent invalid repeater ID %q (ignored)\n", peer.ID, remote, string(data[6:14]))
					return nil
				}

				log.Errorf("peer %d@%s deauthenticated us; re-authenticating\n", peer.ID, remote)
				peer.Status = AuthNone
				return h.handleAuth(peer)

			case len(data) == 15 && bytes.Equal(data[:7], RepeaterPong):
				if !h.checkRepeaterID(data[7:]) {
					log.Warningf("peer %d@%s sent invalid repeater ID %q (ignored)\n", peer.ID, remote, string(data[6:14]))
					return nil
				}
				peer.Last.PongReceived = time.Now()
				break

			case len(data) == 10 && bytes.Equal(data[:6], MasterNAK):
				if !h.checkRepeaterID(data[6:]) {
					log.Warningf("peer %d@%s sent invalid repeater ID %q (ignored)\n", peer.ID, remote, string(data[6:14]))
					return nil
				}
				log.Errorf("peer %d@%s sent NAK; re-establishing link\n", peer.ID, remote)
				peer.Status = AuthNone
				return h.handleAuth(peer)

			default:
				log.Warningf("peer %d@%s sent unexpected packet (status=%s):\n", peer.ID, remote, peer.Status.String())
				log.Debug(hex.Dump(data))
				break
			}
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
	h.rxtx.Lock()
	defer h.rxtx.Unlock()

	// Record last received time
	h.last = time.Now()

	// Offload packet to handle callback
	if peer.PacketReceived != nil {
		return peer.PacketReceived(h, p)
	}
	if h.pf == nil {
		return errors.New("homebrew: no PacketReceived func defined to handle DMR packet")
	}

	return h.pf(h, p)
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
					switch peer.Status {
					case AuthDone:
						switch {
						case now.Sub(peer.Last.PingReceived) > PingTimeout:
							peer.Status = AuthNone
							log.Errorf("peer %d@%s not requesting to ping; dropping connection", peer.ID, peer.Addr)
							if err := h.WriteToPeer(append(MasterClosing, h.id...), peer); err != nil {
								log.Errorf("peer %d@%s close failed: %v\n", peer.ID, peer.Addr, err)
							}
							break
						}
						break
					}
				} else {
					switch peer.Status {
					case AuthNone, AuthBegin:
						switch {
						case now.Sub(peer.Last.PacketReceived) > AuthTimeout:
							log.Errorf("peer %d@%s not responding to login; retrying\n", peer.ID, peer.Addr)
							if err := h.handleAuth(peer); err != nil {
								log.Errorf("peer %d@%s retry failed: %v\n", peer.ID, peer.Addr, err)
							}
							break
						}

					case AuthDone:
						switch {
						case now.Sub(peer.Last.PongReceived) > PingTimeout:
							peer.Status = AuthNone
							log.Errorf("peer %d@%s not responding to ping; trying to re-establish connection", peer.ID, peer.Addr)
							if err := h.WriteToPeer(append(RepeaterClosing, h.id...), peer); err != nil {
								log.Errorf("peer %d@%s close failed: %v\n", peer.ID, peer.Addr, err)
							}
							if err := h.handleAuth(peer); err != nil {
								log.Errorf("peer %d@%s retry failed: %v\n", peer.ID, peer.Addr, err)
							}
							break

						case now.Sub(peer.Last.PingSent) > PingInterval:
							peer.Last.PingSent = now
							if err := h.WriteToPeer(append(MasterPing, h.id...), peer); err != nil {
								log.Errorf("peer %d@%s ping failed: %v\n", peer.ID, peer.Addr, err)
							}
							break
						}
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
	p, err := ParseData(data)
	if err == nil {
		p.RepeaterID = h.Config.ID
	}
	return p, err
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
	id, err := strconv.ParseUint(string(data), 16, 32)
	if err != nil {
		return 0, err
	}
	return uint32(id), nil
}

// Interface compliance check
var _ dmr.Repeater = (*Homebrew)(nil)

func packRepeaterID(id uint32) []byte {
	return []byte(fmt.Sprintf("%08X", id))
}

// BuildData converts DMR packet format to Homebrew packet format.
func BuildData(p *dmr.Packet, repeaterID uint32) []byte {
	var data = make([]byte, 53)
	copy(data[:4], DMRData)
	data[4] = p.Sequence
	data[5] = uint8(p.SrcID >> 16)
	data[6] = uint8(p.SrcID >> 8)
	data[7] = uint8(p.SrcID)
	data[8] = uint8(p.DstID >> 16)
	data[9] = uint8(p.DstID >> 8)
	data[10] = uint8(p.DstID)
	data[11] = uint8(repeaterID >> 24)
	data[12] = uint8(repeaterID >> 16)
	data[13] = uint8(repeaterID >> 8)
	data[14] = uint8(repeaterID)
	data[15] = p.Timeslot | (p.CallType << 1)
	data[16] = uint8(p.StreamID >> 24)
	data[17] = uint8(p.StreamID >> 16)
	data[18] = uint8(p.StreamID >> 8)
	data[19] = uint8(p.StreamID)
	copy(data[20:], p.Data)

	switch p.DataType {
	case dmr.VoiceBurstB, dmr.VoiceBurstC, dmr.VoiceBurstD, dmr.VoiceBurstE, dmr.VoiceBurstF:
		data[15] |= (0x00 << 2)
		data[15] |= (p.DataType - dmr.VoiceBurstA) << 4
		break
	case dmr.VoiceBurstA:
		data[15] |= (0x01 << 2)
		break
	default:
		data[15] |= (0x02 << 2)
		data[15] |= (p.DataType) << 4
	}

	return data
}

// ParseData converts Homebrew packet format to DMR packet format.
func ParseData(data []byte) (*dmr.Packet, error) {
	if len(data) != 53 {
		return nil, fmt.Errorf("homebrew: expected 53 data bytes, got %d", len(data))
	}

	var p = &dmr.Packet{
		Sequence:   data[4],
		SrcID:      uint32(data[5])<<16 | uint32(data[6])<<8 | uint32(data[7]),
		DstID:      uint32(data[8])<<16 | uint32(data[9])<<8 | uint32(data[10]),
		RepeaterID: uint32(data[11])<<24 | uint32(data[12])<<16 | uint32(data[13])<<8 | uint32(data[14]),
		Timeslot:   (data[15] >> 0) & 0x01,
		CallType:   (data[15] >> 1) & 0x01,
		StreamID:   uint32(data[16])<<24 | uint32(data[17])<<16 | uint32(data[18])<<8 | uint32(data[19]),
	}
	p.SetData(data[20:])

	switch (data[15] >> 2) & 0x03 {
	case 0x00, 0x01: // voice (B-F), voice sync (A)
		p.DataType = dmr.VoiceBurstA + (data[15] >> 4)
		break
	case 0x02: // data sync
		p.DataType = (data[15] >> 4)
		break
	default: // unknown/unused
		return nil, errors.New("homebrew: unexpected frame type 0b11")
	}

	return p, nil
}
