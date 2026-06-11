package main

import (
	"encoding/binary"
	"errors"
	"io"
)

const (
	Swapflag  uint8 = 0
	Storeflag uint8 = 1
	Fetchflag uint8 = 2
)

type Request struct {
	Flag  uint8
	Fkey  uint32
	Skey  uint32
	Value []byte
}

func Parserequest(r io.Reader) (*Request, error) {
	tempFlag := make([]byte, 1)
	if _, err := io.ReadFull(r, tempFlag); err != nil {
		return nil, err
	}
	flag := tempFlag[0]
	req := &Request{Flag: flag}

	switch flag {
	case Storeflag:
		buff := make([]byte, 8)
		if _, err := io.ReadFull(r, buff); err != nil {
			return nil, err
		}
		req.Skey = binary.BigEndian.Uint32(buff[0:4])
		vlen := binary.BigEndian.Uint32(buff[4:8])

		vbuf := make([]byte, vlen)
		if _, err := io.ReadFull(r, vbuf); err != nil {
			return nil, err
		}
		req.Value = vbuf

	case Fetchflag:
		buff := make([]byte, 4)
		if _, err := io.ReadFull(r, buff); err != nil {
			return nil, err
		}
		req.Fkey = binary.BigEndian.Uint32(buff[0:4])

	case Swapflag:
		buff := make([]byte, 12)
		if _, err := io.ReadFull(r, buff); err != nil {
			return nil, err
		}
		req.Fkey = binary.BigEndian.Uint32(buff[0:4])
		req.Skey = binary.BigEndian.Uint32(buff[4:8])
		vlen := binary.BigEndian.Uint32(buff[8:12])

		vbuf := make([]byte, vlen)
		if _, err := io.ReadFull(r, vbuf); err != nil {
			return nil, err
		}
		req.Value = vbuf

	default:
		return nil, errors.New("request error: request flag not found")
	}
	return req, nil
}
