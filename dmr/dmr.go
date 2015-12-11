package dmr

const (
	PayloadBits                 = 98 + 10 + 48 + 10 + 98
	PayloadSize                 = 33
	InfoHalfBits                = 98
	InfoBits                    = 2 * InfoHalfBits
	SlotTypeHalfBits            = 10
	SlotTypeBits                = 2 * SlotTypeHalfBits
	SignalBits                  = 48
	SyncBits                    = SignalBits
	VoiceHalfBits               = 108
	VoiceBits                   = 2 * VoiceHalfBits
	EMBHalfBits                 = 8
	EMBBits                     = 2 * EMBHalfBits
	EMBSignallingLCFragmentBits = 32
)
