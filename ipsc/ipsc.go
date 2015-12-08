package ipsc

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"time"
)

type Network struct {
	Disabled                   bool
	RadioID                    uint32
	AliveTimer                 time.Duration
	MaxMissed                  int
	IPSCMode                   string
	PeerOperDisabled           bool
	TS1LinkDisabled            bool
	TS2LinkDisabled            bool
	CSBKCall                   bool
	RepeaterCallMonitoring     bool `yaml:"rcm"`
	ConsoleApplicationDisabled bool
	XNLCall                    bool
	XNLMaster                  bool
	DataCall                   bool
	VoiceCall                  bool
	MasterPeer                 bool
	AuthKey                    string
	Master                     string
	Listen                     string
}

type IPSC struct {
	Network *Network
	Dump    bool

	authKey []byte
	local   struct {
		addr    *net.UDPAddr
		radioID []byte
		mode    byte
		flags   []byte
		tsFlags []byte
	}
	master struct {
		addr    *net.UDPAddr
		radioID []byte
		mode    byte
		flags   []byte
		status  struct {
			connected            bool
			peerList             bool
			keepAliveSent        int
			keepAliveMissed      int
			keepAliveOutstanding int
			keepAliveReceived    int
			keepAliveRXTime      int
		}
	}
	conn *net.UDPConn
}

func New(network *Network) (*IPSC, error) {
	c := &IPSC{
		Network: network,
	}
	c.local.radioID = make([]byte, 4)
	c.local.flags = make([]byte, 4)
	c.master.radioID = make([]byte, 4)
	c.master.flags = make([]byte, 4)

	binary.BigEndian.PutUint32(c.local.radioID, c.Network.RadioID)

	var err error

	if c.Network.AuthKey != "" {
		if c.authKey, err = hex.DecodeString(c.Network.AuthKey); err != nil {
			return nil, err
		}
	}
	if c.Network.AliveTimer == 0 {
		c.Network.AliveTimer = time.Second * 5
	}
	if !c.Network.PeerOperDisabled {
		c.local.mode |= FlagPeerOperational
	}
	switch c.Network.IPSCMode {
	case "analog":
		c.local.mode |= FlagPeerModeAnalog
	case "", "digital":
		c.local.mode |= FlagPeerModeDigital
	case "none":
		c.local.mode &= 0xff ^ MaskPeerMode
	default:
		return nil, fmt.Errorf("unknown IPSCMode %q", c.Network.IPSCMode)
	}
	if c.Network.TS1LinkDisabled {
		c.local.mode |= FlagIPSCTS1Off
	} else {
		c.local.mode |= FlagIPSCTS1On
	}
	if c.Network.TS2LinkDisabled {
		c.local.mode |= FlagIPSCTS2Off
	} else {
		c.local.mode |= FlagIPSCTS2On
	}

	if c.Network.CSBKCall {
		c.local.flags[2] |= FlagCSBKMessage
	}
	if c.Network.RepeaterCallMonitoring {
		c.local.flags[2] |= FlagRepeaterCallMonitoring
	}
	if !c.Network.ConsoleApplicationDisabled {
		c.local.flags[2] |= FlagConsoleApplication
	}

	if c.Network.XNLCall {
		c.local.flags[3] |= FlagXNLStatus
		if c.Network.XNLMaster {
			c.local.flags[3] |= FlagXNLMaster
		} else {
			c.local.flags[3] |= FlagXNLSlave
		}
	}
	if len(c.Network.AuthKey) > 0 {
		c.local.flags[3] |= FlagPacketAuthenticated
	}
	if c.Network.DataCall {
		c.local.flags[3] |= FlagDataCall
	}
	if c.Network.VoiceCall {
		c.local.flags[3] |= FlagVoiceCall
	}
	if c.Network.MasterPeer {
		c.local.flags[3] |= FlagMasterPeer
	}

	if c.Network.Listen == "" {
		c.Network.Listen = ":62030"
	}
	if c.local.addr, err = net.ResolveUDPAddr("udp", c.Network.Listen); err != nil {
		return nil, err
	}
	if c.Network.Master != "" {
		if c.master.addr, err = net.ResolveUDPAddr("udp", c.Network.Master); err != nil {
			return nil, err
		}
	}

	c.local.tsFlags = append([]byte{c.local.mode}, c.local.flags...)

	return c, nil
}

func (c *IPSC) Run() error {
	var err error
	if c.conn, err = net.ListenUDP("udp", c.local.addr); err != nil {
		return err
	}

	go c.peerMaintenance()

	for {
		var peer *net.UDPAddr
		var n int
		var b = make([]byte, 512)
		if n, peer, err = c.conn.ReadFromUDP(b); err != nil {
			log.Printf("error reading from %s: %v\n", peer, err)
			continue
		}

		if c.Dump {
			c.dump(peer, b[:n])
		}
		if !c.authenticate(b[:n]) {
			log.Printf("authentication failed, dropping packet from %s\n", peer)
			continue
		}
	}

	return nil
}

func (c *IPSC) authenticate(data []byte) bool {
	if c.authKey == nil || len(c.authKey) == 0 {
		return true
	}

	payload := c.payload(data)
	hash := data[len(data)-10:]
	mac := hmac.New(sha1.New, c.authKey)
	mac.Write(payload)
	return hmac.Equal(hash, mac.Sum(nil))
}

func (c *IPSC) payload(data []byte) []byte {
	if c.authKey == nil || len(c.authKey) == 0 {
		return data
	}
	return data[:len(data)-10]
}

func (c *IPSC) dump(addr *net.UDPAddr, data []byte) {
	if len(data) < 7 {
		fmt.Printf("%d bytes of unreadable data from %s:\n", len(data), addr)
		fmt.Printf(hex.Dump(data))
		return
	}

	fmt.Printf("%d bytes of data %s:\n", len(data), addr)
	fmt.Printf(hex.Dump(data))

	packetType := data[0]
	peerID := binary.BigEndian.Uint32(data[1:5])
	seq := data[5:6]

	switch packetType {
	case CallConfirmation:
		fmt.Println("call confirmation:")
	case TextMessageAck:
		fmt.Println("text message acknowledgement:")
	case CallMonStatus:
		fmt.Println("call monitor status:")
	case CallMonRepeat:
		fmt.Println("call monitor repeater:")
	case CallMonNACK:
		fmt.Println("call monitor nack:")
	case XCMPXNLControl:
		fmt.Println("XCMP/XNL control message:")
	case GroupVoice:
		fmt.Println("group voice:")
	case PVTVoice:
		fmt.Println("PVT voice:")
	case GroupData:
		fmt.Println("group data:")
	case PVTData:
		fmt.Println("PVT data:")
	case RPTWakeUp:
		fmt.Println("RPT wake up:")
	case UnknownCollision:
		fmt.Println("unknown collision:")
	case MasterRegistrationRequest:
		fmt.Println("master registration request:")
	case MasterRegistrationReply:
		fmt.Println("master registration reply:")
	case PeerListRequest:
		fmt.Println("peer list request:")
	case PeerListReply:
		fmt.Println("peer list reply:")
	case PeerRegistrationRequest:
		fmt.Println("peer registration request:")
	case PeerRegistrationReply:
		fmt.Println("peer registration reply:")
	case MasterAliveRequest:
		fmt.Println("master alive request:")
	case MasterAliveReply:
		fmt.Println("master alive reply:")
	case PeerAliveRequest:
		fmt.Println("peer alive request:")
	case PeerAliveReply:
		fmt.Println("peer alive reply:")
	case DeregistrationRequest:
		fmt.Println("de-registration request:")
	case DeregistrationReply:
		fmt.Println("de-registration reply:")
	default:
		fmt.Printf("unknown packet type 0x%02x:\n", packetType)
	}
	fmt.Printf("\tpeer id: %d\n", peerID)
	fmt.Printf("\tsequence: %v\n", seq)
}

func (c *IPSC) hashedPacket(key, data []byte) []byte {
	if key == nil || len(key) == 0 {
		return data
	}

	mac := hmac.New(sha1.New, key)
	mac.Write(data)

	hash := make([]byte, 20)
	hex.Encode(hash, mac.Sum(nil))

	return append(data, hash...)
}

func (c *IPSC) peerMaintenance() {
	for {
		var p []byte
		if c.master.status.connected {
			log.Println("sending keep-alive to master")
			r := []byte{MasterAliveRequest}
			r = append(r, c.local.radioID...)
			r = append(r, c.local.tsFlags...)
			r = append(r, []byte{LinkTypeIPSC, Version17, LinkTypeIPSC, Version16}...)
			p = c.hashedPacket(c.authKey, r)

		} else {
			log.Println("registering with master")
			r := []byte{MasterRegistrationRequest}
			r = append(r, c.local.radioID...)
			r = append(r, c.local.tsFlags...)
			r = append(r, []byte{LinkTypeIPSC, Version17, LinkTypeIPSC, Version16}...)
			p = c.hashedPacket(c.authKey, r)
		}

		if err := c.sendToMaster(p); err != nil {
			log.Fatalf("error sending registration request to master: %v\n", err)
			return
		}

		time.Sleep(c.Network.AliveTimer)
	}
}

func (c *IPSC) sendToMaster(data []byte) error {
	if c.Dump {
		c.dump(c.master.addr, data)
	}
	for len(data) > 0 {
		n, err := c.conn.WriteToUDP(data, c.master.addr)
		if err != nil {
			return err
		}
		data = data[n:]
	}
	return nil
}
