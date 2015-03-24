package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/catalyzeio/taptun/pktutil"
)

func main() {
	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		fmt.Printf("Error: missing pcap file argument\n")
		return
	}

	p, err := pktutil.OpenPcap(args[0])
	if err != nil {
		fmt.Printf("Error: could not read file: %s\n", err)
		return
	}
	defer p.Close()

	for {
		pkt, err := p.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("Error: could not read packet: %s\n", err)
			break
		}
		fmt.Printf("%s - %d bytes", pkt.Timestamp, len(pkt.Data))
		if pkt.Truncated {
			fmt.Printf(" (truncated)")
		}
		fmt.Printf("\n")
		// XXX the remaining code does not do any bounds-checking
		fmt.Printf("Layer 2\n")
		fmt.Printf("    Src  MAC: %s\n", pktutil.MACSource(pkt.Data))
		fmt.Printf("    Dest MAC: %s\n", pktutil.MACDestination(pkt.Data))
		payload := pktutil.MACPayload(pkt.Data)
		if pktutil.IsIPv4(payload) {
			fmt.Printf("Layer 3 - IPv4\n")
			fmt.Printf("    Src  IP: %s\n", pktutil.IPv4Source(payload))
			fmt.Printf("    Dest IP: %s\n", pktutil.IPv4Destination(payload))
			data := pktutil.IPv4Payload(payload)
			fmt.Printf("Data\n    %x\n", data)
		} else {
			fmt.Printf("Payload\n    %x\n", payload)
		}
		fmt.Printf("\n")
	}
}
