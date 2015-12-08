package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"pd0mz/dmr/homebrew"

	"gopkg.in/yaml.v2"
)

func rc() *homebrew.RepeaterConfiguration {
	return &homebrew.RepeaterConfiguration{
		Callsign:    "PI1BOL",
		RepeaterID:  2043044,
		RXFreq:      433787500,
		TXFreq:      438787500,
		TXPower:     5,
		ColorCode:   1,
		Latitude:    52.296786,
		Longitude:   4.595454,
		Height:      12,
		Location:    "Hillegom, ZH, NL",
		Description: fmt.Sprintf("%s go-dmr", homebrew.Version),
		URL:         "https://github.com/pd0mz",
	}
}

var (
	callType = map[homebrew.CallType]string{
		homebrew.GroupCall: "group",
		homebrew.UnitCall:  "unit",
	}
	frameType = map[homebrew.FrameType]string{
		homebrew.Voice:           "voice",
		homebrew.VoiceSync:       "voice sync",
		homebrew.DataSync:        "data sync",
		homebrew.UnusedFrameType: "unused (should not happen)",
	}
)

func dump(d *homebrew.Data) {
	fmt.Println("DMR data:")
	fmt.Printf("\tsequence: %d\n", d.Sequence)
	fmt.Printf("\ttarget..: %d -> %d\n", d.SrcID, d.DstID)
	fmt.Printf("\trepeater: %d\n", d.RepeaterID)
	fmt.Printf("\tslot....: %d\n", d.Slot())
	fmt.Printf("\tcall....: %s\n", callType[d.CallType()])
	fmt.Printf("\tframe...: %s\n", frameType[d.FrameType()])
	switch d.FrameType() {
	case homebrew.DataSync:
		fmt.Printf("\tdatatype: %d\n", d.DataType())
	case homebrew.Voice, homebrew.VoiceSync:
		fmt.Printf("\tsequence: %c (voice)\n", 'A'+d.DataType())
	}
	fmt.Printf("\tdump....:\n")
	fmt.Println(hex.Dump(d.DMR[:]))
}

func main() {
	configFile := flag.String("config", "dmr-homebrew.yaml", "configuration file")
	flag.Parse()

	f, err := os.Open(*configFile)
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

	repeater, err := homebrew.New(network, rc, dump)
	if err != nil {
		panic(err)
	}

	repeater.Dump = true
	panic(repeater.Run())
}
