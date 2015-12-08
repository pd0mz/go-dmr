package ipsc

const (
	// IPSC Version Information
	Version14  byte = 0x00
	Version15  byte = 0x00
	Version15A byte = 0x00
	Version16  byte = 0x01
	Version17  byte = 0x02
	Version18  byte = 0x02
	Version19  byte = 0x03
	Version22  byte = 0x04

	// Known IPSC Message Types
	CallConfirmation          byte = 0x05 // Confirmation FROM the recipient of a confirmed call.
	TextMessageAck            byte = 0x54 // Doesn't seem to mean success, though. This code is sent success or failure
	CallMonStatus             byte = 0x61 //  |
	CallMonRepeat             byte = 0x62 //  | Exact meaning unknown
	CallMonNACK               byte = 0x63 //  |
	XCMPXNLControl            byte = 0x70 // XCMP/XNL control message
	GroupVoice                byte = 0x80
	PVTVoice                  byte = 0x81
	GroupData                 byte = 0x83
	PVTData                   byte = 0x84
	RPTWakeUp                 byte = 0x85 // Similar to OTA DMR "wake up"
	UnknownCollision          byte = 0x86 // Seen when two dmrlinks try to transmit at once
	MasterRegistrationRequest byte = 0x90 // FROM peer TO master
	MasterRegistrationReply   byte = 0x91 // FROM master TO peer
	PeerListRequest           byte = 0x92 // From peer TO master
	PeerListReply             byte = 0x93 // From master TO peer
	PeerRegistrationRequest   byte = 0x94 // Peer registration request
	PeerRegistrationReply     byte = 0x95 // Peer registration reply
	MasterAliveRequest        byte = 0x96 // FROM peer TO master
	MasterAliveReply          byte = 0x97 // FROM master TO peer
	PeerAliveRequest          byte = 0x98 // Peer keep alive request
	PeerAliveReply            byte = 0x99 // Peer keep alive reply
	DeregistrationRequest     byte = 0x9a // Request de-registration from system
	DeregistrationReply       byte = 0x9b // De-registration reply

	// Link Type Values
	LinkTypeIPSC byte = 0x04
)

var AnyPeerRequired = map[byte]bool{
	GroupVoice:            true,
	PVTVoice:              true,
	GroupData:             true,
	PVTData:               true,
	CallMonStatus:         true,
	CallMonRepeat:         true,
	CallMonNACK:           true,
	XCMPXNLControl:        true,
	RPTWakeUp:             true,
	DeregistrationRequest: true,
}

var PeerRequired = map[byte]bool{
	PeerAliveRequest:        true,
	PeerAliveReply:          true,
	PeerRegistrationRequest: true,
	PeerRegistrationReply:   true,
}

var MasterRequired = map[byte]bool{
	PeerListReply:    true,
	MasterAliveReply: true,
}

var UserGenerated = map[byte]bool{
	GroupVoice: true,
	PVTVoice:   true,
	GroupData:  true,
	PVTData:    true,
}
