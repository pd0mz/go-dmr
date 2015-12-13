package main

import (
	"crypto/rand"
	"fmt"
)

func main() {
	var raw = make([]byte, 4)
	rand.Read(raw)
	fmt.Printf("raw = %02x%02x%02x%02x\n", raw[0], raw[1], raw[2], raw[3])

	for i := 0; i < 4*8; i++ {
		b := i / 8
		o := byte(7 - i%8)
		fmt.Printf("%02x b=%d, o=%d, r=%d\n", raw[b], b, o, raw[b]>>o)
	}
	for i := 0; i < 4*8; i++ {
		b := i / 8
		o := byte(7 - i%8)
		if (raw[b]>>o)&0x01 == 0x01 {
			fmt.Print("1")
		} else {
			fmt.Print("0")
		}
	}
	fmt.Println("")
}
