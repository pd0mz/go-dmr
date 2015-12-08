package main

import (
	"flag"
	"io/ioutil"
	"os"
	"pd0mz/dmr/ipsc"

	"gopkg.in/yaml.v2"
)

func main() {
	configFile := flag.String("config", "dmr.yaml", "configuration file")
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

	network := &ipsc.Network{}
	if err := yaml.Unmarshal(d, network); err != nil {
		panic(err)
	}

	repeater, err := ipsc.New(network)
	if err != nil {
		panic(err)
	}

	repeater.Dump = true
	panic(repeater.Run())
}
