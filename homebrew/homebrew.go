// Package homebrew implements the Home Brew DMR IPSC protocol
package homebrew

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

var (
	Version    = "20151208"
	SoftwareID = fmt.Sprintf("%s:go-dmr:%s", runtime.GOOS, Version)
	PackageID  = fmt.Sprintf("%s:go-dmr:%s-%s", runtime.GOOS, Version, runtime.GOARCH)
)

// RepeaterConfiguration holds information about the current repeater. It
// should be returned by a callback in the implementation, returning actual
// information about the current repeater status.
type RepeaterConfiguration struct {
	Callsign    string
	RepeaterID  uint32
	RXFreq      uint32
	TXFreq      uint32
	TXPower     uint8
	ColorCode   uint8
	Latitude    float32
	Longitude   float32
	Height      uint16
	Location    string
	Description string
	URL         string
}

// Bytes returns the configuration as bytes.
func (r *RepeaterConfiguration) Bytes() []byte {
	return []byte(r.String())
}

// String returns the configuration as string.
func (r *RepeaterConfiguration) String() string {
	if r.ColorCode < 1 {
		r.ColorCode = 1
	}
	if r.ColorCode > 15 {
		r.ColorCode = 15
	}
	if r.TXPower > 99 {
		r.TXPower = 99
	}

	var lat = fmt.Sprintf("%-08f", r.Latitude)
	if len(lat) > 8 {
		lat = lat[:8]
	}
	var lon = fmt.Sprintf("%-09f", r.Longitude)
	if len(lon) > 9 {
		lon = lon[:9]
	}

	var b = "RPTC"
	b += fmt.Sprintf("%-8s", r.Callsign)
	b += fmt.Sprintf("%08x", r.RepeaterID)
	b += fmt.Sprintf("%09d", r.RXFreq)
	b += fmt.Sprintf("%09d", r.TXFreq)
	b += fmt.Sprintf("%02d", r.TXPower)
	b += fmt.Sprintf("%02d", r.ColorCode)
	b += lat
	b += lon
	b += fmt.Sprintf("%03d", r.Height)
	b += fmt.Sprintf("%-20s", r.Location)
	b += fmt.Sprintf("%-20s", r.Description)
	b += fmt.Sprintf("%-124s", r.URL)
	b += fmt.Sprintf("%-40s", SoftwareID)
	b += fmt.Sprintf("%-40s", PackageID)
	return b
}

type configFunc func() *RepeaterConfiguration

// CallType reflects the DMR data frame call type.
type CallType byte

const (
	GroupCall CallType = iota
	UnitCall
)

// FrameType reflects the DMR data frame type.
type FrameType byte

const (
	Voice FrameType = iota
	VoiceSync
	DataSync
	UnusedFrameType
)

// Frame is a frame of DMR data.
type Frame struct {
	Signature  [4]byte
	Sequence   byte
	SrcID      uint32
	DstID      uint32
	RepeaterID uint32
	Flags      byte
	StreamID   uint32
	DMR        [33]byte
}

func (f *Frame) CallType() CallType {
	return CallType((f.Flags >> 1) & 0x01)
}

func (f *Frame) DataType() byte {
	return f.Flags >> 4
}

func (f *Frame) FrameType() FrameType {
	return FrameType((f.Flags >> 2) & 0x03)
}

func (f *Frame) Slot() int {
	return int(f.Flags&0x01) + 1
}

func ParseFrame(data []byte) (*Frame, error) {
	if len(data) != 53 {
		return nil, errors.New("invalid packet length")
	}

	f := &Frame{}
	copy(f.Signature[:], data[:4])
	f.Sequence = data[4]
	f.SrcID = binary.BigEndian.Uint32(append([]byte{0x00}, data[5:7]...))
	f.DstID = binary.BigEndian.Uint32(append([]byte{0x00}, data[8:10]...))
	f.RepeaterID = binary.BigEndian.Uint32(data[11:15])
	f.Flags = data[15]
	f.StreamID = binary.BigEndian.Uint32(data[16:20])
	copy(f.DMR[:], data[20:])

	return f, nil
}

type streamFunc func(*Frame)

type authStatus byte

const (
	authNone authStatus = iota
	authBegin
	authDone
	authFail
)

type Network struct {
	AuthKey  string
	Local    string
	LocalID  uint32
	Master   string
	MasterID uint32
}

type Link struct {
	Dump    bool
	config  configFunc
	stream  streamFunc
	network *Network
	conn    *net.UDPConn
	authKey []byte
	local   struct {
		addr *net.UDPAddr
		id   []byte
	}
	master struct {
		addr      *net.UDPAddr
		id        []byte
		status    authStatus
		secret    []byte
		keepalive struct {
			outstanding uint32
			sent        uint64
		}
	}
}

// New starts a new DMR repeater using the Home Brew protocol.
func New(network *Network, cf configFunc, sf streamFunc) (*Link, error) {
	if cf == nil {
		return nil, errors.New("config func can't be nil")
	}

	link := &Link{
		network: network,
		config:  cf,
		stream:  sf,
	}

	var err error
	if strings.HasPrefix(network.AuthKey, "0x") {
		if link.authKey, err = hex.DecodeString(network.AuthKey[2:]); err != nil {
			return nil, err
		}
	} else {
		link.authKey = []byte(network.AuthKey)
	}
	if network.Local == "" {
		network.Local = "0.0.0.0:62030"
	}
	if network.LocalID == 0 {
		return nil, errors.New("missing localid")
	}
	link.local.id = []byte(fmt.Sprintf("%08x", network.LocalID))
	if link.local.addr, err = net.ResolveUDPAddr("udp", network.Local); err != nil {
		return nil, err
	}
	if network.Master == "" {
		return nil, errors.New("no master address configured")
	}
	if link.master.addr, err = net.ResolveUDPAddr("udp", network.Master); err != nil {
		return nil, err
	}

	return link, nil
}

// Run starts the datagram receiver and logs the repeater in with the master.
func (l *Link) Run() error {
	var err error

	if l.conn, err = net.ListenUDP("udp", l.local.addr); err != nil {
		return err
	}

	go l.login()

	for {
		var (
			n    int
			peer *net.UDPAddr
			data = make([]byte, 512)
		)
		if n, peer, err = l.conn.ReadFromUDP(data); err != nil {
			log.Printf("dmr/homebrew: error reading from %s: %v\n", peer, err)
			continue
		}

		go l.parse(peer, data[:n])
	}

	return nil
}

// Send data to an UDP address using the repeater datagram socket.
func (l *Link) Send(addr *net.UDPAddr, data []byte) error {
	for len(data) > 0 {
		n, err := l.conn.WriteToUDP(data, addr)
		if err != nil {
			return err
		}
		data = data[n:]
	}
	return nil
}

func (l *Link) login() {
	var previous = authDone
	for l.master.status != authFail {
		var p []byte

		if l.master.status != previous {
			switch l.master.status {
			case authNone:
				log.Printf("dmr/homebrew: logging in as %d\n", l.network.LocalID)
				p = append(RepeaterLogin, l.local.id...)

			case authBegin:
				log.Printf("dmr/homebrew: authenticating as %d\n", l.network.LocalID)
				p = append(RepeaterKey, l.local.id...)

				hash := sha256.New()
				hash.Write(l.master.secret)
				hash.Write(l.authKey)

				p = append(p, []byte(hex.EncodeToString(hash.Sum(nil)))...)

			case authDone:
				config := l.config().Bytes()
				fmt.Printf(hex.Dump(config))
				log.Printf("dmr/homebrew: logged in, sending %d bytes of repeater configuration\n", len(config))

				if err := l.Send(l.master.addr, config); err != nil {
					log.Printf("dmr/homebrew: send(%s) failed: %v\n", l.master.addr, err)
					return
				}
				l.keepAlive()
				return

			case authFail:
				log.Println("dmr/homebrew: login failed")
				return
			}
			if p != nil {
				l.Send(l.master.addr, p)
			}
			previous = l.master.status
		} else {
			log.Println("dmr/homebrew: waiting for master to respond in login sequence...")
			time.Sleep(time.Second)
		}
	}
}

func (l *Link) keepAlive() {
	for {
		atomic.AddUint32(&l.master.keepalive.outstanding, 1)
		atomic.AddUint64(&l.master.keepalive.sent, 1)
		var p = append(MasterPing, l.local.id...)
		if err := l.Send(l.master.addr, p); err != nil {
			log.Printf("dmr/homebrew: send(%s) failed: %v\n", l.master.addr, err)
			return
		}
		time.Sleep(time.Minute)
	}
}

func (l *Link) parse(addr *net.UDPAddr, data []byte) {
	size := len(data)

	switch l.master.status {
	case authNone:
		if bytes.Equal(data, DMRData) {
			return
		}
		if size < 14 {
			return
		}
		packet := data[:6]
		repeater, err := hex.DecodeString(string(data[6:14]))
		if err != nil {
			log.Println("dmr/homebrew: unexpected login reply from master")
			l.master.status = authFail
			break
		}

		switch {
		case bytes.Equal(packet, MasterNAK):
			log.Printf("dmr/homebrew: login refused by master %d\n", repeater)
			l.master.status = authFail
			break
		case bytes.Equal(packet, MasterACK):
			log.Printf("dmr/homebrew: login accepted by master %d\n", repeater)
			l.master.secret = data[14:]
			l.master.status = authBegin
			break
		default:
			log.Printf("dmr/homebrew: unexpected login reply from master %d\n", repeater)
			l.master.status = authFail
			break
		}

	case authBegin:
		if bytes.Equal(data, DMRData) {
			return
		}
		if size < 14 {
			log.Println("dmr/homebrew: unexpected login reply from master")
			l.master.status = authFail
			break
		}
		packet := data[:6]
		repeater, err := hex.DecodeString(string(data[6:14]))
		if err != nil {
			log.Println("dmr/homebrew: unexpected login reply from master")
			l.master.status = authFail
			break
		}

		switch {
		case bytes.Equal(packet, MasterNAK):
			log.Printf("dmr/homebrew: authentication refused by master %d\n", repeater)
			l.master.status = authFail
			break
		case bytes.Equal(packet, MasterACK):
			log.Printf("dmr/homebrew: authentication accepted by master %d\n", repeater)
			l.master.status = authDone
			break
		default:
			log.Printf("dmr/homebrew: unexpected authentication reply from master %d\n", repeater)
			l.master.status = authFail
			break
		}

	case authDone:
		if len(data) < 4 {
			return
		}
		switch {
		case bytes.Equal(data[:4], DMRData):
			if l.stream == nil {
				return
			}
			frame, err := ParseFrame(data)
			if err != nil {
				log.Printf("error parsing DMR data: %v\n", err)
				return
			}
			l.stream(frame)
		}
	}
}
