package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"gopkg.in/yaml.v2"

	"github.com/tehmaze/go-dmr/bit"
	"github.com/tehmaze/go-dmr/dmr"
	"github.com/tehmaze/go-dmr/dmr/repeater"
	"github.com/tehmaze/go-dmr/homebrew"
	"github.com/tehmaze/go-dmr/ipsc"
)

var (
	bursttypes = map[uint32]string{
		0: "pi header",
		1: "voice lc header",
		2: "terminator with lc",
		3: "csbk",
		4: "csbk",
		5: "appended mbc",
		6: "data header",
		7: "rate ½ data continuation",
		8: "rate ¾ data continuation",
		9: "idle",
	}
	headertypes = []string{
		"UDT ", "Resp", "UDat", "CDat", "Hdr4", "Hdr5", "Hdr6", "Hdr7",
		"Hdr8", "Hdr9", "Hdr10", "Hdr11", "Hdr12", "DSDa", "RSDa", "Prop",
	}
	saptypes = []string{
		"UDT", "1", "TCP HC", "UDP HC", "IP Pkt", "ARP Pkt", "6", "7",
		"8", "Prop Pkt", "Short Data", "11", "12", "13", "14", "15",
	}
	fidMap = [256]byte{
		0x01, 0x00, 0x00, 0x00, 0x02, 0x03, 0x04, 0x05,
		0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x07, 0x00, 0x00, 0x08, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x09, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x0a, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x0b, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x0c, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x0d, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0e,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	fidName = map[uint8]string{
		0:  "Unknown",
		1:  "Standard",
		2:  "Flyde Micro",
		3:  "PROD-EL SPA",
		4:  "Motorola Connect+",
		5:  "RADIODATA GmbH",
		6:  "Hyteria (8)",
		7:  "Motorola Capacity+",
		8:  "EMC S.p.A (19)",
		9:  "EMC S.p.A (28)",
		10: "Radio Activity Srl (51)",
		11: "Radio Activity Srl (60)",
		12: "Tait Electronics",
		13: "Hyteria (104)",
		14: "Vertex Standard",
	}
)

func dumpFCLO(payload []byte, fid uint8) {
	fmt.Printf("    fid: %s (%d)\n", fidName[fid], fid)
	if fid == 0 || fid == 16 { // MotoTRBO Capacity+
		fmt.Printf("    fclo: -\n")
	}
}

func dumpDataHeader(header []byte) {
	dh, err := dmr.ParseDataHeader(header, false)
	if err != nil {
		fmt.Printf("  data header error: %v\n", err)
		return
	}
	dc := dh.CommonHeader()
	fmt.Printf("    service accesspt: %d\n", dc.ServiceAccessPoint)
	fmt.Printf("    src id -> dst id: %d -> %d\n", dc.SrcID, dc.DstID)
	switch d := dh.(type) {
	case dmr.UDTDataHeader:
		fmt.Printf("    format..........: %s (%#02x)\n", dmr.UDTFormatName[d.Format], d.Format)
		fmt.Printf("    pad nibble......: %d\n", d.PadNibble)
		fmt.Printf("    appended blocks.: %d\n", d.AppendedBlocks)
		fmt.Printf("    supplementary f.: %b\n", d.SupplementaryFlag)
		fmt.Printf("    opcode..........: %d\n", d.OPCode)
	case dmr.UnconfirmedDataHeader:
		fmt.Printf("    pad octet count.: %d\n", d.PadOctetCount)
		fmt.Printf("    full message....: %b\n", d.FullMessage)
		fmt.Printf("    blocks to follow: %d\n", d.BlocksToFollow)
		fmt.Printf("    fragment seq.no.: %d\n", d.FragmentSequenceNumber)
	}
}

func dumpData(data bit.Bits) {
	var out = make([][]byte, 7)
	for y := 0; y < 7; y++ {
		out[y] = make([]byte, 80)
		for x := 0; x < 80; x++ {
			if x == 0 || x == 18 || x == 21 || x == 30 || x == 33 || x == 51 {
				out[y][x] = '|'
			} else {
				out[y][x] = ' '
			}
		}
	}
	// Info first half
	for i := 0; i < 98; i++ {
		var x = 1 + (i % 16)
		var y = (i / 16)
		if x > 8 {
			x++
		}
		if data[i] > 0 {
			out[y][x] = '1'
		} else {
			out[y][x] = '0'
		}
	}
	// Slot type first half
	for i := 98; i < 108; i++ {
		var x = 19 + ((i - 98) % 2)
		var y = (i - 98) / 2
		if data[i] > 0 {
			out[y][x] = '1'
		} else {
			out[y][x] = '0'
		}
	}
	// SYNC
	for i := 108; i < 156; i++ {
		var x = 22 + ((i - 108) % 8)
		var y = (i - 108) / 8
		if data[i] > 0 {
			out[y][x] = '1'
		} else {
			out[y][x] = '0'
		}
	}
	// Slot type second half
	for i := 156; i < 166; i++ {
		var x = 31 + ((i - 156) % 2)
		var y = (i - 156) / 2
		if data[i] > 0 {
			out[y][x] = '1'
		} else {
			out[y][x] = '0'
		}
	}
	// Info second half
	for i := 166; i < 264; i++ {
		var x = 34 + ((i - 166) % 16)
		var y = ((i - 166) / 16)
		if x > 41 {
			x++
		}
		if data[i] > 0 {
			out[y][x] = '1'
		} else {
			out[y][x] = '0'
		}
	}
	fmt.Println("|------INFO1------|ST|--SYNC--|ST|------INFO2------|")
	for _, row := range out {
		fmt.Println(string(row))
	}
}

func dumpSync(p *ipsc.Packet, desc string) {
	sync := dmr.ExtractSyncBits(p.PayloadBits)
	patt := dmr.SyncPattern(sync)
	fmt.Printf("dmr[%d->%d]: ts%d got %s:\n", p.SrcID, p.DstID, p.Timeslot+1, desc)
	fmt.Printf("  sync pattern: %s\n", dmr.SyncPatternName[patt])
	if patt == dmr.SyncPatternUnknown {
		fmt.Print(hex.Dump(sync.Bytes()))
		for i, b := range sync {
			sync[i] = b ^ 1
		}
		fmt.Print(hex.Dump(sync.Bytes()))
	}

	/*
		slot := dmr.ExtractSlotType(p.PayloadBits)
		var (
			codeword  uint32
			bursttype uint32
			payload   = make([]byte, 12)
		)

		codeword = (uint32(slot[0]) << 11) | (uint32(slot[1]) << 3) | uint32(slot[2])>>5
		bursttype = uint32(slot[0]) & 7
		fmt.Printf("codeword: %#08x, bursttype: %#04x\n", codeword, bursttype)
		fec.Golay_23_12_Correct(&codeword)
		fmt.Printf("codeword: %#08x (after correcting)\n", codeword)
		codeword &= 0x0f
		fmt.Printf("codeword: %#08x (after masking)\n", codeword)
		bursttype ^= codeword
		var errors int
		if bursttype&1 > 0 {
			errors++
		}
		if bursttype&2 > 0 {
			errors++
		}
		if bursttype&4 > 0 {
			errors++
		}
		if bursttype&8 > 0 {
			errors++
		}
		bursttype = codeword
		//fmt.Printf("%d errors detected, burstype: %08x (%d)\n", errors, bursttype, bursttype)
		fmt.Printf("  burst type: %s (%d), %d errors\n", bursttypes[bursttype], bursttype, errors)

		if bursttype < 7 {
			if err := bptc.Process(dmr.ExtractInfoBits(p.PayloadBits), payload); err != nil {
				fmt.Printf("  payload error: %v\n", err)
				return
			}
			fmt.Printf("  payload: (%d bytes)\n", len(payload))
			fmt.Print("  " + hex.Dump(payload))
		}

		fid := payload[1]
		fidm := fidMap[fid]

		switch bursttype {
		case 1, 2:
			dumpFCLO(payload, fidm)
		case 6:
			dumpDataHeader(payload)
		}

		/*
			if err := fec.Golay_20_8_Check(slot); err != nil {
				fmt.Printf("%v\n", err)
				fmt.Print(hex.Dump(slot.Bytes()))
				return
			}

			cc := (slot[0] << 3) | (slot[1] << 2) | (slot[2] << 1) | slot[3]
			dt := (slot[4] << 3) | (slot[5] << 2) | (slot[6] << 1) | slot[7]
			fmt.Printf("cc: %d, data type: %d\n", cc, dt)
	*/
}

func rc() *homebrew.RepeaterConfiguration {
	return &homebrew.RepeaterConfiguration{
		Callsign:    "PI1BOL",
		RepeaterID:  2043044,
		RXFreq:      0,
		TXFreq:      0,
		TXPower:     0,
		ColorCode:   1,
		Latitude:    52.296786,
		Longitude:   4.595454,
		Height:      12,
		Location:    "Hillegom, ZH, NL",
		Description: fmt.Sprintf("%s go-dmr", homebrew.Version),
		URL:         "https://github.com/tehmaze",
	}
}

func dumpRaw(raw []byte) {
	fmt.Printf("dump raw frame of %d bytes\n", len(raw))
	p, err := homebrew.ParseData(raw)
	if err != nil {
		fmt.Printf("  parse error: %v\n", err)
		return
	}
	dumpPacket(p)
}

func dumpPacket(p *ipsc.Packet) {
	fmt.Print(p.Dump())
	dumpData(p.PayloadBits)

	switch p.SlotType {
	case ipsc.VoiceLCHeader:
		dumpSync(p, "voice LC header")
	case ipsc.TerminatorWithLC:
		dumpSync(p, "terminator with LC")
	case ipsc.CSBK:
		dumpSync(p, "CSBK")
	case ipsc.VoiceDataA:
		dumpSync(p, "voice A")
	case ipsc.VoiceDataB:
		dumpSync(p, "voice B")
	case ipsc.VoiceDataC:
		dumpSync(p, "voice C")
	case ipsc.VoiceDataD:
		dumpSync(p, "voice D")
	case ipsc.VoiceDataE:
		dumpSync(p, "voice E")
	case ipsc.VoiceDataF:
		dumpSync(p, "voice F")
	}

	fmt.Print("\n---\n\n")
}

func main() {
	dumpFile := flag.String("dump", "", "dump file")
	liveFile := flag.String("live", "", "live configuration file")
	showRaw := flag.Bool("raw", false, "show raw frames")
	flag.Parse()

	r := repeater.New()
	if *liveFile != "" {
		log.Printf("going in live mode using %q\n", *liveFile)
		f, err := os.Open(*liveFile)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		d, err := ioutil.ReadAll(f)
		if err != nil {
			panic(err)
		}

		network := &homebrew.Network{}
		if err := yaml.Unmarshal(d, network); err != nil {
			panic(err)
		}

		protocol, err := homebrew.New(network, rc, r.Stream)
		if err != nil {
			panic(err)
		}

		protocol.Dump = true
		panic(protocol.Run())

	} else {

		i, err := os.Open(*dumpFile)
		if err != nil {
			log.Fatal(err)
		}
		defer i.Close()

		var raw = make([]byte, 53)
		for {
			if _, err := i.Read(raw); err != nil {
				panic(err)
			}
			if *showRaw {
				fmt.Println("raw packet:")
				fmt.Print(hex.Dump(raw))
			}

			//dumpRaw(raw)

			p, err := homebrew.ParseData(raw)
			if err != nil {
				fmt.Printf("  parse error: %v\n", err)
				continue
			}
			r.Stream(p)

			// Skip newline in recording
			if _, err := i.Seek(1, 1); err != nil {
				panic(err)
			}
		}
	}
}
