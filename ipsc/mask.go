package ipsc

type Mask uint8

// IPSC mask values

// Linking status
/*
	Byte 1 - BIT FLAGS:

		xx.. .... = Peer Operational (01 only known valid value)
		..xx .... = Peer MODE: 00 - No Radio, 01 - Analog, 10 - Digital
		.... xx.. = IPSC Slot 1: 10 on, 01 off
		.... ..xx = IPSC Slot 2: 10 on, 01 off
*/
const (
	FlagPeerOperational = 0x40
	MaskPeerMode        = 0x30
	FlagPeerModeAnalog  = 0x10
	FlagPeerModeDigital = 0x20
	MaskIPSCTS1         = 0x0c
	MaskIPSCTS2         = 0x03
	FlagIPSCTS1On       = 0x08
	FlagIPSCTS1Off      = 0x04
	FlagIPSCTS2On       = 0x02
	FlagIPSCTS2Off      = 0x01
)

// Service flags
/*
	Byte 1 - 0x00  	= Unknown
	Byte 2 - 0x00	= Unknown
	Byte 3 - BIT FLAGS:

		x... .... = CSBK Message
		.x.. .... = Repeater Call Monitoring
		..x. .... = 3rd Party "Console" Application
		...x xxxx = Unknown - default to 0
*/
const (
	FlagCSBKMessage            = 0x80
	FlagRepeaterCallMonitoring = 0x40
	FlagConsoleApplication     = 0x20
)

/*
	Byte 4 = BIT FLAGS:

		x... .... = XNL Connected (1=true)
		.x.. .... = XNL Master Device
		..x. .... = XNL Slave Device
		...x .... = Set if packets are authenticated
		.... x... = Set if data calls are supported
		.... .x.. = Set if voice calls are supported
		.... ..x. = Unknown - default to 0
		.... ...x = Set if master
*/
const (
	FlagXNLStatus           = 0x80
	FlagXNLMaster           = 0x40
	FlagXNLSlave            = 0x20
	FlagPacketAuthenticated = 0x10
	FlagDataCall            = 0x08
	FlagVoiceCall           = 0x04
	FlagMasterPeer          = 0x01
)

// Timeslot call and status byte
/*
	Byte 17 of Group and Private Voice/Data Packets

		..x.. ....TS Value (0=TS1, 1=TS2)
		.x... ....TS In Progress/End (0=In Progress, 1=End)

	Possible values: 0x00=TS1, 0x20=TS2, 0x40=TS1 End, 0x60=TS2 End
*/

// RTP mask values
/*
	Bytes 1 and 2 of the RTP header are bit-fields, the rest are at least
	one byte long, and do not need masked.
*/
const (
	// Byte 1
	RTPVersionMask Mask = 0xc0
	RTPPadMask     Mask = 0x20
	RTPExtMask     Mask = 0x10
	RTPCSICMask    Mask = 0x0f
	// Byte 2
	RTPMRKRMask    Mask = 0x80
	RTPPayTypeMask Mask = 0xf7
)
