package main

import (
	"encoding/binary"
	"errors"
	"io"
	"fmt"
)

const (
    Swapflag uint8 = 0
    Storeflag uint8 = 1
    Fetchflag uint8 = 2
)

func executestore(w io.ReadWriter, key uint32, value []byte) error{
	vlen := uint32(len(value))

	header := make([]byte,9)
	header[0] = Storeflag
	binary.BigEndian.PutUint32(header[1:5],key)
	binary.BigEndian.PutUint32(header[5:9],vlen)

	w.Write(header)
	w.Write(value)

	ack := make([]byte,1)
	io.ReadFull(w,ack)
	if ack[0]!=1{
		return errors.New("Error: Store action failed")
	}
	return nil
}

func executefetch(w io.ReadWriter, key uint32) ([]byte, error){
	// Header size is correct
	header := make([]byte, 5)
	header[0] = Fetchflag
	binary.BigEndian.PutUint32(header[1:5], key)
	if _, err := w.Write(header); err != nil {
		return nil, err
	}

	status := make([]byte, 1)
	if _, err := io.ReadFull(w, status); err != nil {
		// If the server connection drops, we handle the error safely.
		return nil, fmt.Errorf("connection read error on status flag: %v", err)
	}
	
	if status[0] == 0 {
		return nil, errors.New("error: block cache miss")
	}

	sizeBuf := make([]byte, 4)
	if _, err := io.ReadFull(w, sizeBuf); err != nil {
		return nil, fmt.Errorf("connection read error on payload size: %v", err)
	}
	size := binary.BigEndian.Uint32(sizeBuf)
	
	buff := make([]byte, size)
	if _, err := io.ReadFull(w, buff); err != nil {
		return nil, fmt.Errorf("connection read error on payload body: %v", err)
	}
	
	return buff, nil
}

func executeswap(w io.ReadWriter, fkey uint32, skey uint32, svalue []byte) ([]byte,error){
	vlen := uint32(len(svalue))

	header := make([]byte,13)
	header[0] = Swapflag
	binary.BigEndian.PutUint32(header[1:5],fkey)
	binary.BigEndian.PutUint32(header[5:9],skey)
	binary.BigEndian.PutUint32(header[9:13],vlen)
	w.Write(header)
	w.Write(svalue)

	status := make([]byte, 1)
	io.ReadFull(w,status)
	if status[0] != 1{
		return nil, errors.New("Error: Cache miss")
	}

	ackvalue := make([]byte,4)
	io.ReadFull(w,ackvalue)
	size := binary.BigEndian.Uint32(ackvalue)
	buff := make([]byte, size)
	io.ReadFull(w,buff)
	return buff,nil
}

