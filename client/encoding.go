package main

import (
	"encoding/binary"
	"errors"
	"io"
)

// Shared Constants
const (
	Swapflag  uint8 = 0
	Storeflag uint8 = 1
	Fetchflag uint8 = 2
)

func executestore(w io.ReadWriter, key uint32, value []byte) error {
	vlen := uint32(len(value))

	header := make([]byte, 9)
	header[0] = Storeflag
	binary.BigEndian.PutUint32(header[1:5], key)
	binary.BigEndian.PutUint32(header[5:9], vlen)

	w.Write(header)
	w.Write(value)

	ack := make([]byte, 1)
	if _, err := io.ReadFull(w, ack); err != nil {
		return err
	}
	if ack[0] != 1 {
		return errors.New("Error: Store action failed")
	}
	return nil
}

func executefetch(w io.ReadWriter, key uint32) ([]byte, error) {
	header := make([]byte, 5)
	header[0] = Fetchflag
	binary.BigEndian.PutUint32(header[1:5], key)
	if _, err := w.Write(header); err != nil {
		return nil, err
	}

	status := make([]byte, 1)
	if _, err := io.ReadFull(w, status); err != nil {
		return nil, err
	}

	if status[0] == 0 {
		return nil, errors.New("error: block cache miss")
	}

	var size uint32
	if err := binary.Read(w, binary.BigEndian, &size); err != nil {
		return nil, err
	}

	buff := make([]byte, size)
	if _, err := io.ReadFull(w, buff); err != nil {
		return nil, err
	}

	return buff, nil
}

func executeswap(w io.ReadWriter, fkey uint32, skey uint32, svalue []byte) ([]byte, error) {
	vlen := uint32(len(svalue))

	header := make([]byte, 13)
	header[0] = Swapflag
	binary.BigEndian.PutUint32(header[1:5], fkey)
	binary.BigEndian.PutUint32(header[5:9], skey)
	binary.BigEndian.PutUint32(header[9:13], vlen)
	w.Write(header)
	w.Write(svalue)

	status := make([]byte, 1)
	if _, err := io.ReadFull(w, status); err != nil {
		return nil, err
	}
	if status[0] != 1 {
		return nil, errors.New("Error: Cache miss")
	}

	var size uint32
	if err := binary.Read(w, binary.BigEndian, &size); err != nil {
		return nil, err
	}
	buff := make([]byte, size)
	if _, err := io.ReadFull(w, buff); err != nil {
		return nil, err
	}
	return buff, nil
}