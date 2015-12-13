package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"gopkg.in/yaml.v2"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"github.com/gordonklaus/portaudio"
	"github.com/tehmaze/go-dmr/bit"
	"github.com/tehmaze/go-dmr/dmr/repeater"
	"github.com/tehmaze/go-dmr/homebrew"
	"github.com/tehmaze/go-dmr/ipsc"
	"github.com/tehmaze/go-dsd"
)

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

func main() {
	dumpFile := flag.String("dump", "", "dump file")
	pcapFile := flag.String("pcap", "", "PCAP file")
	liveFile := flag.String("live", "", "live configuration file")
	showRaw := flag.Bool("raw", false, "show raw frames")
	audioTS := flag.Int("audiots", 0, "play audio from time slot (1 or 2)")
	flag.Parse()

	if *audioTS > 2 {
		log.Fatalf("invalid time slot %d\n", *audioTS)
		return
	}

	r := repeater.New()

	if *audioTS > 0 {
		ambeframe := make(chan float32)

		vs := dsd.NewAMBEVoiceStream(3)
		r.VoiceFrameFunc = func(p *ipsc.Packet, bits bit.Bits) {
			var in = make([]byte, len(bits))
			for i, b := range bits {
				in[i] = byte(b)
			}

			samples, err := vs.Decode(in)
			if err != nil {
				log.Printf("error decoding AMBE3000 frame: %v\n", err)
				return
			}
			for _, sample := range samples {
				ambeframe <- sample
			}
		}

		portaudio.Initialize()
		defer portaudio.Terminate()
		h, err := portaudio.DefaultHostApi()
		if err != nil {
			panic(err)
		}

		p := portaudio.LowLatencyParameters(nil, h.DefaultOutputDevice)
		p.SampleRate = 8000
		p.Output.Channels = 1
		stream, err := portaudio.OpenStream(p, func(out []float32) {
			for i := range out {
				out[i] = <-ambeframe
			}
		})
		if err != nil {
			log.Printf("error streaming: %v\n", err)
			return
		}
		defer stream.Close()
		if err := stream.Start(); err != nil {
			log.Printf("error streaming: %v\n", err)
			return
		}
	}

	switch {
	case *liveFile != "":
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

	case *dumpFile != "":
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

	case *pcapFile != "":
		var (
			handle *pcap.Handle
			err    error
		)

		if handle, err = pcap.OpenOffline(*pcapFile); err != nil {
			panic(err)
		}
		defer handle.Close()

		dec := gopacket.DecodersByLayerName["Ethernet"]
		source := gopacket.NewPacketSource(handle, dec)
		for packet := range source.Packets() {
			raw := packet.ApplicationLayer().Payload()
			if *showRaw {
				fmt.Println("raw packet:")
				fmt.Print(hex.Dump(raw))
			}

			p, err := homebrew.ParseData(raw)
			if err != nil {
				fmt.Printf("  parse error: %v\n", err)
				continue
			}
			r.Stream(p)
		}
	}
}
