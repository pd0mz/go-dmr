// Package homebrew implements the Home Brew DMR IPSC protocol
package homebrew

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/tehmaze/go-dmr/ipsc"
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

// ConfigFunc returns an actual RepeaterConfiguration instance when called.
// This is used by the DMR repeater to poll for current configuration,
// statistics and metrics.
type ConfigFunc func() *RepeaterConfiguration

// CallType reflects the DMR data frame call type.
type CallType byte

const (
	GroupCall CallType = iota
	UnitCall
)

// StreamFunc is called by the DMR repeater when a DMR data frame is received.
type StreamFunc func(*ipsc.Packet)

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

type packet struct {
	addr *net.UDPAddr
	data []byte
}

type Link struct {
	Dump    bool
	config  ConfigFunc
	stream  StreamFunc
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
func New(network *Network, cf ConfigFunc, sf StreamFunc) (*Link, error) {
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

// Close stops the socket and stops the runner
func (l *Link) Close() error {
	if l.conn == nil {
		return errors.New("dmr/homebrew: link not open")
	}
	if l.master.addr != nil {
		l.Send(l.master.addr, append(RepeaterClosing, l.local.id...))
	}

	return l.conn.Close()
}

// Run starts the datagram receiver and logs the repeater in with the master.
func (l *Link) Run() error {
	var err error

	if l.conn, err = net.ListenUDP("udp", l.local.addr); err != nil {
		return err
	}

	queue := make(chan packet)
	go l.login()
	go l.parse(queue)

receiving:
	for {
		var (
			n    int
			peer *net.UDPAddr
			data = make([]byte, 512)
		)
		if n, peer, err = l.conn.ReadFromUDP(data); err != nil {
			if peer == nil {
				break receiving
			}
			log.Printf("dmr/homebrew: error reading from %s: %v\n", peer, err)
			continue
		}

		queue <- packet{peer, data[:n]}
	}

	// Because we close it in .Close()
	if strings.HasSuffix(err.Error(), ": use of closed network connection") {
		return nil
	}
	return err
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
				if l.Dump {
					fmt.Printf(hex.Dump(config))
				}
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

func (l *Link) parse(queue <-chan packet) {
	for {
		select {
		case p := <-queue:
			size := len(p.data)
			if size < 4 {
				continue
			}

			switch l.master.status {
			case authNone:
				if bytes.Equal(p.data[:4], DMRData) {
					return
				}
				if size < 14 {
					return
				}
				packet := p.data[:6]
				repeater, err := hex.DecodeString(string(p.data[6:14]))
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
					l.master.secret = p.data[14:]
					l.master.status = authBegin
					break
				default:
					log.Printf("dmr/homebrew: unexpected login reply from master %d\n", repeater)
					l.master.status = authFail
					break
				}

			case authBegin:
				if bytes.Equal(p.data[:4], DMRData) {
					return
				}
				if size < 14 {
					log.Println("dmr/homebrew: unexpected login reply from master")
					l.master.status = authFail
					break
				}
				packet := p.data[:6]
				repeater, err := hex.DecodeString(string(p.data[6:14]))
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
				if l.Dump {
					fmt.Printf("received from %s:\n", p.addr)
					fmt.Print(hex.Dump(p.data))
				}
				switch {
				case bytes.Equal(p.data[:4], DMRData):
					l.parseDMR(p.data)
				}
			}
		}
	}
}

func (l *Link) parseDMR(d []byte) {
	// If we have no stream callback, don't even bother to decode the DMR data frame.
	if l.stream == nil {
		return
	}

	var (
		p   *ipsc.Packet
		err error
	)
	if p, err = ParseData(d); err != nil {
		log.Printf("dmr/homebrew: error parsing DMRD frame: %v\n", err)
		return
	}
	l.stream(p)
}
