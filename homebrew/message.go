package homebrew

// Messages as documented by DL5DI, G4KLX and DG1HT, see
// http://download.prgm.org/dl5di-soft/dmrplus/documentation/Homebrew-Repeater/DMRplus%20IPSC%20Protocol%20for%20HB%20repeater%20(20150726).pdf
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
