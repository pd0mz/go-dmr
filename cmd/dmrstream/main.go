package main

/*
#cgo LDFLAGS: -lshout
#include "shout/shout.h"
#include <stdlib.h>
*/
import "C"

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"gopkg.in/yaml.v2"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"github.com/tehmaze/go-dmr/bit"
	"github.com/tehmaze/go-dmr/dmr/repeater"
	"github.com/tehmaze/go-dmr/homebrew"
	"github.com/tehmaze/go-dmr/ipsc"
	"github.com/tehmaze/go-dsd"
)

const (
	Timeslot1 uint8 = iota
	Timeslot2
)

var (
	VoiceFrameDuration = time.Millisecond * 60
	VoiceSyncDuration  = time.Millisecond * 360
	UserMap            = map[uint32]string{}
)

type Protocol interface {
	Close() error
	Run() error
}

type Config struct {
	Repeater *homebrew.RepeaterConfiguration
	Link     map[string]*Link
	User     string
}

type Shout struct {
	Host        string
	Port        uint
	User        string
	Password    string
	Mount       string
	Format      int
	Protocol    int
	Description string
	Genre       string

	// wrap the native C struct
	shout    *C.struct_shout
	metadata *C.struct_shout_metadata_t

	stream chan []byte
}

func (s *Shout) getError() error {
	errstr := C.GoString(C.shout_get_error(s.shout))
	return errors.New("shout: " + errstr)
}

func (s *Shout) init() error {
	if s.shout != nil {
		return nil
	}

	s.shout = C.shout_new()
	s.stream = make(chan []byte)
	return s.update()
}

func (s *Shout) update() error {
	// set hostname
	p := C.CString(s.Host)
	C.shout_set_host(s.shout, p)
	C.free(unsafe.Pointer(p))

	// set port
	C.shout_set_port(s.shout, C.ushort(s.Port))

	// set username
	p = C.CString(s.User)
	C.shout_set_user(s.shout, p)
	C.free(unsafe.Pointer(p))

	// set password
	p = C.CString(s.Password)
	C.shout_set_password(s.shout, p)
	C.free(unsafe.Pointer(p))

	// set mount point
	p = C.CString(s.Mount)
	C.shout_set_mount(s.shout, p)
	C.free(unsafe.Pointer(p))

	// set description
	p = C.CString(s.Description)
	C.shout_set_description(s.shout, p)
	C.free(unsafe.Pointer(p))

	// set genre
	p = C.CString(s.Genre)
	C.shout_set_genre(s.shout, p)
	C.free(unsafe.Pointer(p))

	// set format
	C.shout_set_format(s.shout, C.uint(s.Format))

	// set protocol
	C.shout_set_protocol(s.shout, C.uint(s.Protocol))

	return nil
}

func (s *Shout) Close() error {
	if s.shout != nil {
		C.shout_free(s.shout)
		s.shout = nil
	}
	return nil
}

func (s *Shout) Open() error {
	if err := s.init(); err != nil {
		return err
	}

	errno := int(C.shout_open(s.shout))
	if errno != 0 {
		return s.getError()
	}

	return nil
}

func (s *Shout) Stream(data []byte) error {
	if s.shout == nil {
		return errors.New("shout: stream not open")
	}

	ptr := (*C.uchar)(&data[0])
	C.shout_send(s.shout, ptr, C.size_t(len(data)))
	errno := int(C.shout_get_errno(s.shout))
	if errno != 0 {
		return s.getError()
	}

	C.shout_sync(s.shout)
	return nil
}

func (s *Shout) UpdateDescription(desc string) {
	ptr := C.CString(desc)
	C.shout_set_description(s.shout, ptr)
	C.free(unsafe.Pointer(ptr))
}

func (s *Shout) UpdateGenre(genre string) {
	ptr := C.CString(genre)
	C.shout_set_genre(s.shout, ptr)
	C.free(unsafe.Pointer(ptr))
}

func (s *Shout) UpdateMetadata(mname string, mvalue string) {
	md := C.shout_metadata_new()
	ptr1 := C.CString(mname)
	ptr2 := C.CString(mvalue)
	C.shout_metadata_add(md, ptr1, ptr2)
	C.free(unsafe.Pointer(ptr1))
	C.free(unsafe.Pointer(ptr2))
	C.shout_set_metadata(s.shout, md)
	C.shout_metadata_free(md)
}

type Link struct {
	Disable bool

	// Supported protocols
	Homebrew *homebrew.Network
	PCAP     *PCAPProtocol

	// Shout streams
	Transcode string
	TS1Stream *Shout
	TS2Stream *Shout
}

type Stream struct {
	Timeslot  uint8
	Repeater  string
	Transcode string // Transcoder binary/script
	Shout     *Shout // Shout server details
	Buffer    chan float32
	Samples   []float32

	pipe      io.WriteCloser // Our transcoder input
	running   bool
	connected bool
	retry     int
}

func NewStream(ts uint8, repeater string, sh *Shout, transcode string) *Stream {
	if sh.Description == "" {
		sh.Description = fmt.Sprintf("DMR Repeater %s (TS%d)", strings.ToUpper(repeater), ts+1)
	}
	if sh.Genre == "" {
		sh.Genre = "ham"
	}
	return &Stream{
		Timeslot:  ts,
		Repeater:  repeater,
		Transcode: transcode,
		Shout:     sh,
		Buffer:    make(chan float32),
		Samples:   make([]float32, 8000),
	}
}

func (s *Stream) Close() error {
	if s.running {
		s.running = false
		if s.pipe != nil {
			return s.pipe.Close()
		}
	}
	return nil
}

func (s *Stream) Run() error {
	var err error

	log.Printf("dmr/stream: connecting to icecast server %s:%d%s as %s\n",
		s.Shout.Host, s.Shout.Port, s.Shout.Mount, s.Shout.User)
	if err = s.Shout.Open(); err != nil {
		return err
	}
	s.connected = true
	s.Shout.UpdateDescription(fmt.Sprintf("DMR repeater link to %s (TS%d)", s.Repeater, s.Timeslot+1))

	log.Println("dmr/stream: setting up transcoder pipe")
	cmnd := strings.Split(s.Transcode, " ")
	pipe := exec.Command(cmnd[0], cmnd[1:]...)
	enc, err := pipe.StdinPipe()
	if err != nil {
		return err
	}
	defer enc.Close()
	s.pipe = enc

	out, err := pipe.StdoutPipe()
	if err != nil {
		log.Printf("dmr/stream: error connecting to stream output: %v\n", err)
		return err
	}
	defer out.Close()

	// Connect stderr
	pipe.Stderr = os.Stderr

	if err := pipe.Start(); err != nil {
		return err
	}

	// Spawn goroutine that deals with new audio from the transcode process
	go func(out io.Reader) {
		var buf = make([]byte, 1024)
		for {
			if _, err := out.Read(buf); err != nil {
				log.Printf("dmr/stream: error reading from stream: %v\n", err)
				s.Close()
				return
			}
			var err = s.Shout.Stream(buf)
			for err != nil {
				log.Printf("dmr/stream: error streaming: %v\n", err)
				s.Close()

				s.retry++
				if s.retry > 15 {
					log.Printf("dmr/stream: retry limit exceeded\n")
					return
				}

				time.Sleep(time.Second * time.Duration(3*s.retry))
				log.Printf("dmr/stream: connecting to icecast server %s:%d%s as %s\n",
					s.Shout.Host, s.Shout.Port, s.Shout.Mount, s.Shout.User)
				err = s.Shout.Open()
			}

			s.retry = 0
		}
	}(out)

	var i uint32
	s.running = true
	for s.running {
		// Ensure that we *always* have new data (even be it silence) within the duration of a voice frame
		select {
		case sample := <-s.Buffer:
			s.Samples[i] = sample
			i++
		case <-time.After(VoiceFrameDuration):
			log.Printf("dmr/stream: filling silence, timeout after %s\n", VoiceFrameDuration)
			for ; i < 8000; i++ {
				s.Samples[i] = 0
			}
		}

		if i >= 8000 {
			log.Printf("dmr/stream: writing %d samples to encoder\n", i)
			var buffer = make([]byte, 4)
			for _, sample := range s.Samples {
				binary.BigEndian.PutUint32(buffer, math.Float32bits(sample))
				if _, err := enc.Write(buffer); err != nil {
					log.Printf("dmr/stream: error writing to encoder: %v\n", err)
					return err
				}
			}

			i = 0
		}
	}

	return nil
}

func (s *Stream) UpdateMetadata(repeater string, src, dst uint32) {
	if s.connected {
		log.Printf("dmr/stream: updating meta data to %s and %d -> %d\n", repeater, src, dst)
		s.Shout.UpdateMetadata("description", fmt.Sprintf("Repeater %s", strings.ToUpper(repeater)))
		s.Shout.UpdateMetadata("artist", fmt.Sprintf("Repeater %s", strings.ToUpper(repeater)))

		var (
			dstName = strconv.Itoa(int(dst))
			srcName = strconv.Itoa(int(src))
		)
		if name, ok := UserMap[dst]; ok {
			dstName = name
		}
		if name, ok := UserMap[src]; ok {
			srcName = name
		}

		s.Shout.UpdateMetadata("song", fmt.Sprintf("TS%d [%s -> %s]", s.Timeslot+1, srcName, dstName))
	}
}

func (s *Stream) Write(sample float32) {
	s.Buffer <- sample
}

type Samples struct {
	data             []float32
	size, rptr, wptr int
}

func NewSamples(size int) *Samples {
	return &Samples{
		data: make([]float32, size),
		size: size,
	}
}

func (s *Samples) Read() float32 {
	d := s.data[s.rptr]
	s.data[s.rptr] = 0
	s.rptr++
	if s.rptr == s.size {
		s.rptr = 0
	}
	return d
}

func (s *Samples) Write(sample float32) {
	s.data[s.wptr] = sample
	s.wptr++
	if s.wptr == s.size {
		s.wptr = 0
	}
}

type PCAPProtocol struct {
	Filename string
	DumpRaw  bool
	Stream   homebrew.StreamFunc
	closed   bool
}

func (pp *PCAPProtocol) Close() error {
	pp.closed = true
	return nil
}

func (pp *PCAPProtocol) Run() error {
	var (
		handle *pcap.Handle
		err    error
	)

	if handle, err = pcap.OpenOffline(pp.Filename); err != nil {
		return err
	}
	defer handle.Close()

	dec := gopacket.DecodersByLayerName["Ethernet"]
	source := gopacket.NewPacketSource(handle, dec)
	for packet := range source.Packets() {
		raw := packet.ApplicationLayer().Payload()
		if pp.DumpRaw {
			fmt.Println("raw packet:")
			fmt.Print(hex.Dump(raw))
		}

		p, err := homebrew.ParseData(raw)
		if err != nil {
			fmt.Printf("  parse error: %v\n", err)
			continue
		}
		if pp.Stream != nil {
			pp.Stream(p)
		}
		if pp.closed {
			break
		}
	}

	return nil
}

func init() {
	C.shout_init()
}

func main() {
	configFile := flag.String("config", "", "configuration file")
	amplify := flag.Float64("amplify", 25.0, "audio amplify rate")
	verbose := flag.Bool("verbose", false, "be verbose")
	enabled := flag.String("enable", "", "comma separated list of enabled links (overrides config)")
	disabled := flag.String("disable", "", "comma separated list of disabled links (overrides config)")
	flag.Parse()

	overrides := map[string]bool{}
	for _, call := range strings.Split(*enabled, ",") {
		overrides[call] = true
	}
	for _, call := range strings.Split(*disabled, ",") {
		overrides[call] = false
	}

	log.Printf("using configuration file %q\n", *configFile)
	f, err := os.Open(*configFile)
	if err != nil {
		log.Fatalf("failed to open %q: %v\n", *configFile, err)
		panic(err)
	}
	defer f.Close()

	d, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatalf("failed to read %q: %v\n", *configFile, err)
		return
	}

	config := &Config{}
	if err := yaml.Unmarshal(d, config); err != nil {
		log.Fatalf("failed to parse %q: %v\n", *configFile, err)
		return
	}

	if config.User != "" {
		uf, err := os.Open(config.User)
		if err != nil {
			log.Fatalf("failed to open %q: %v\n", config.User, err)
			return
		}
		defer uf.Close()

		scanner := bufio.NewScanner(uf)
		var lines int
		for scanner.Scan() {
			part := strings.Split(string(scanner.Text()), ";")
			if lines > 1 {
				if dmrID, err := strconv.ParseUint(part[2], 10, 32); err == nil {
					UserMap[uint32(dmrID)] = fmt.Sprintf("%s (%s)", part[3], part[1])
				}
			}
			lines++
		}
		if err := scanner.Err(); err != nil {
			log.Fatalf("failed to parse %q: %v\n", config.User, err)
			return
		}
	}

	if len(config.Link) == 0 {
		log.Fatalln("no links configured")
		return
	}

	sm := map[string]map[uint8]*Stream{}
	ps := map[string]Protocol{}

	for call, link := range config.Link {
		status, ok := overrides[call]
		switch {
		case ok:
			if !status {
				log.Printf("link/%s: link disabled, skipping (override)\n", call)
				continue
			}
			log.Printf("link/%s: link enabled (override)\n", call)
		case link.Disable:
			log.Printf("link/%s: link disabled, skipping\n", call)
			continue
		}
		log.Printf("link/%s: configuring link\n", call)

		// Repeater
		r := repeater.New()

		// Protocol
		switch {
		case link.Homebrew != nil:
			log.Printf("link/%s: homebrew protocol, %s\n", call, link.Homebrew.Master)

			rc := func() *homebrew.RepeaterConfiguration {
				return config.Repeater
			}
			proto, err := homebrew.New(link.Homebrew, rc, r.Stream)
			if err != nil {
				log.Fatalf("link/%s: failed to setup protocol: %v\n", call, err)
				return
			}

			ps[call] = proto

		case link.PCAP != nil:
			log.Printf("link/%s: PCAP file %q\n", call, link.PCAP.Filename)
			link.PCAP.Stream = r.Stream

			ps[call] = link.PCAP

		default:
			log.Fatalf("[%s]: unknown or no protocol configured\n", call)
			return
		}

		// Streams
		sm[call] = map[uint8]*Stream{}
		if link.TS1Stream != nil {
			if link.Transcode == "" {
				log.Fatalf("link/%s: TS1 stream defined, but no transcoder\n", call)
			}
			sm[call][Timeslot1] = NewStream(Timeslot1, call, link.TS1Stream, link.Transcode)
		}
		if link.TS2Stream != nil {
			if link.Transcode == "" {
				log.Fatalf("link/%s: TS2 stream defined, but no transcoder\n", call)
			}
			sm[call][Timeslot2] = NewStream(Timeslot1, call, link.TS2Stream, link.Transcode)
		}

		// Setup AMBE voice stream decoder
		var (
			lastsrc, lastdst uint32
			last             = time.Now()
			vs               = dsd.NewAMBEVoiceStream(3)
		)

		// Function that receives decoded AMBE frames as float32 PCM (8kHz mono)
		r.VoiceFrameFunc = func(p *ipsc.Packet, bits bit.Bits) {
			var in = make([]byte, len(bits))
			for i, b := range bits {
				in[i] = byte(b)
			}

			samples, err := vs.Decode(in)
			if err != nil {
				log.Printf("error decode AMBE3000 frames: %v\n", err)
				return
			}
			for _, sample := range samples {
				if stream, ok := sm[call][p.Timeslot]; ok {
					stream.Write(sample * float32(*amplify))

					if lastsrc != p.SrcID || lastdst != p.DstID {
						stream.UpdateMetadata(call, p.SrcID, p.DstID)
						lastsrc = p.SrcID
						lastdst = p.DstID
					}
				}
			}
			if false {
				diff := time.Now().Sub(last)
				if *verbose {
					log.Printf("%s since last voice sync\n", diff)
				}
				if diff < VoiceFrameDuration {
					t := VoiceFrameDuration - diff
					if *verbose {
						log.Printf("delaying %s, last tick was %s ago\n", t, diff)
					}
					time.Sleep(t)
				}
				last = time.Now()
			}
		}
	}

	// Signal handler
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func(signals chan os.Signal) {
		for _ = range signals {
			// Terminate protocols
			for call, p := range ps {
				log.Printf("link/%s: closing\n", call)
				if err := p.Close(); err != nil {
					log.Printf("link/%s: close error: %v\n", call, err)
					continue
				}
			}

			// Terminate streams
			for call, streams := range sm {
				for ts, stream := range streams {
					log.Printf("link/%s: closing stream for TS%d\n", call, ts+1)
					if err := stream.Close(); err != nil {
						log.Printf("link/%s: error closing stream for TS%d: %v\n", call, ts+1, err)
						continue
					}
				}
			}
		}
	}(c)

	wg := &sync.WaitGroup{}
	// Spawn a goroutine for all the protocol runners
	for call, p := range ps {
		wg.Add(1)
		go func(call string, p Protocol, wg *sync.WaitGroup) {
			defer wg.Done()
			if err := p.Run(); err != nil {
				log.Printf("link/%s: error running: %v\n", call, err)
			}
			log.Printf("link/%s: done\n", call)
			delete(ps, call)
		}(call, p, wg)
	}

	// Spawn a goroutine for all the streamers
	for call, streams := range sm {
		if len(streams) == 0 {
			continue
		}
		for ts, stream := range streams {
			log.Printf("link/%s: starting stream for TS%d\n", call, ts+1)
			wg.Add(1)
			go func(call string, stream *Stream, wg *sync.WaitGroup) {
				defer wg.Done()
				if err := stream.Run(); err != nil {
					log.Printf("link/%s: error streaming: %v\n", call, err)
				}
				log.Printf("link/%s: stream done\n", call)
			}(call, stream, wg)
		}
	}

	// Wait for protocols to finish
	log.Println("all routines started, waiting for protocols to finish")
	wg.Wait()

}
