package pktutil

import (
	"net"
)

type Tagging int

// Indicating whether/how a MAC frame is tagged.
// The value is number of bytes taken by tagging.
const (
	NotTagged    Tagging = 0
	Tagged       Tagging = 4
	DoubleTagged Tagging = 8
)

func MACDestination(macFrame []byte) net.HardwareAddr {
	return net.HardwareAddr(macFrame[:6])
}

func MACSource(macFrame []byte) net.HardwareAddr {
	return net.HardwareAddr(macFrame[6:12])
}

func MACTagging(macFrame []byte) Tagging {
	b1, b2 := macFrame[12], macFrame[13]
	// check for TPID in ethertype position
	if b1 == 0x81 && b2 == 0x00 {
		b3, b4 := macFrame[16], macFrame[17]
		if b3 == 0x81 && b4 == 0x00 {
			// literal Q-in-Q tagging
			return DoubleTagged
		}
		// normal 802.1q tagging
		return Tagged
	}
	// older Q-in-Q (non-standard) ethertype values
	if (b1 == 0x91 || b1 == 0x92) && b2 == 0x00 {
		return DoubleTagged
	}
	// newer 802.1ad ethertype value
	if b1 == 0x88 && b2 == 0xA8 {
		return DoubleTagged
	}
	// no 802.1q tagging
	return NotTagged
}

func MACEthertype(macFrame []byte) Ethertype {
	ethertypePos := 12 + MACTagging(macFrame)
	return Ethertype{macFrame[ethertypePos], macFrame[ethertypePos+1]}
}

func MACPayload(macFrame []byte) []byte {
	return macFrame[12+MACTagging(macFrame)+2:]
}

func IsMACBroadcast(addr net.HardwareAddr) bool {
	return addr[0] == 0xFF && addr[1] == 0xFF && addr[2] == 0xFF && addr[3] == 0xFF && addr[4] == 0xFF && addr[5] == 0xFF
}

func IsMACMulticastIPv4(addr net.HardwareAddr) bool {
	return addr[0] == 0x01 && addr[1] == 0x00 && addr[2] == 0x5E
}

func IsMACMulticastIPv6(addr net.HardwareAddr) bool {
	return addr[0] == 0x33 && addr[1] == 0x33
}
