package main

import (
	"encoding/binary"
	"errors"
	"io"
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
	header := make([]byte,5)
	header[0] = Fetchflag
	binary.BigEndian.PutUint32(header[1:5],key)
	w.Write(header)
	
	ackvalue := make([]byte,5)
	io.ReadFull(w,ackvalue)
	if ackvalue[0]!=1{
		return nil,errors.New("Error: Cache miss")
	}
	size := binary.BigEndian.Uint32(ackvalue[1:5])
	buff := make([]byte, size)
	io.ReadFull(w,buff)
	return buff,nil
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

	ackvalue := make([]byte,5)
	io.ReadFull(w,ackvalue)
	if ackvalue[0]!=1{
		return nil,errors.New("Error: Cache miss")
	}
	size := binary.BigEndian.Uint32(ackvalue[1:5])
	buff := make([]byte, size)
	io.ReadFull(w,buff)
	return buff,nil
}

