package pktutil

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"
)

type PcapReader struct {
	file   *os.File
	reader *bufio.Reader
	order  binary.ByteOrder
}

type PcapPacket struct {
	Timestamp time.Time
	Truncated bool
	Data      []byte
}

// Format based on https://wiki.wireshark.org/Development/LibpcapFileFormat

type pcapHeader struct {
	MagicNumber  uint32
	VersionMajor uint16
	VersionMinor uint16
	Zone         int32
	Sigfigs      uint32
	Snaplen      uint32
	Network      linkType
}

type pcapPacketHeader struct {
	TsSec   uint32
	TsUsec  uint32
	InclLen uint32
	OrigLen uint32
}

type linkType uint32

const (
	pcapMagic = 0xA1B2C3D4

	linkTypeEthernet linkType = 1
)

func OpenPcap(fileName string) (*PcapReader, error) {
	success := false
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer func() {
		if !success {
			file.Close()
		}
	}()

	for _, order := range []binary.ByteOrder{binary.BigEndian, binary.LittleEndian} {
		_, err := file.Seek(0, 0)
		if err != nil {
			return nil, err
		}
		reader := bufio.NewReader(file)
		header, err := readHeader(file, order)
		if err != nil {
			return nil, err
		}
		if header != nil {
			if header.VersionMajor != 2 && header.VersionMinor != 4 {
				return nil, fmt.Errorf("unsupported version %d.%d", header.VersionMajor, header.VersionMinor)
			}
			if header.Network != linkTypeEthernet {
				return nil, fmt.Errorf("unsupported link type %d", header.Network)
			}
			success = true
			return &PcapReader{file, reader, order}, nil
		}
	}
	return nil, fmt.Errorf("unsupported file format")
}

func readHeader(reader io.Reader, order binary.ByteOrder) (*pcapHeader, error) {
	header := pcapHeader{}
	err := binary.Read(reader, order, &header)
	if err != nil {
		return nil, err
	}
	if header.MagicNumber != pcapMagic {
		return nil, nil
	}
	return &header, nil
}

func (pr *PcapReader) Close() error {
	return pr.file.Close()
}

func (pr *PcapReader) Read() (*PcapPacket, error) {
	header := pcapPacketHeader{}
	err := binary.Read(pr.reader, pr.order, &header)
	if err != nil {
		return nil, err
	}
	len := header.InclLen
	buff := make([]byte, len)
	_, err = io.ReadFull(pr.reader, buff)
	if err != nil {
		return nil, err
	}
	timestamp := time.Unix(int64(header.TsSec), int64(header.TsUsec*1000))
	totLen := header.OrigLen
	return &PcapPacket{timestamp, len != totLen, buff}, nil
}
