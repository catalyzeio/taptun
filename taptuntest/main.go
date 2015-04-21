package main

import (
	"flag"
	"fmt"

	"github.com/catalyzeio/taptun"
)

var (
	tap      bool
	accessor bool
)

func main() {
	flag.BoolVar(&tap, "tap", true, "whether to create a tap device")
	flag.BoolVar(&accessor, "accessor", false, "whether to use the accessor interface")
	flag.Parse()

	err := read()
	if err != nil {
		fmt.Printf("Error testing tun/tap device: %s\n", err)
	}
}

func read() error {
	var i *taptun.Interface
	var err error
	if tap {
		i, err = taptun.NewTUN("tun%d")
	} else {
		i, err = taptun.NewTAP("tap%d")
	}
	if err != nil {
		return err
	}

	ifName := i.Name()
	fmt.Printf("Created interface %s\n", ifName)

	p := make([]byte, 65536)
	if !accessor {
		for {
			n, err := i.Read(p)
			if err != nil {
				return err
			}
			fmt.Printf("Read %d bytes from interface %s\n", n, ifName)
		}
	} else {
		acc, err := i.Accessor()
		if err != nil {
			return err
		}
		for {
			n, err := acc.Read(p)
			if err != nil {
				return err
			}
			fmt.Printf("Read %d bytes from accessor for %s\n", n, ifName)
		}
	}
}
