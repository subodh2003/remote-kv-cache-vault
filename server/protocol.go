package main

import (
	"encoding/binary"
	"errors"
	"io"
)

const (
    Swapflag = 0
    Storeflag = 1
    Fetchflag = 2
)

type Request struct{
    Flag uint8
    Fkey string
    Skey string
    Value []byte
}

func Parserequest(r io.Reader) (*Request, error){
    tempFlag := make([]byte, 1)
    if _,err:= io.ReadFull(r, tempFlag); err != nil{
        return nil, err
    }
    flag := tempFlag[0]
    req := &Request{Flag: flag}

    switch flag{
    case Storeflag:
        headerbuf := make([]byte, 6)
        if _,err := io.ReadFull(r, headerbuf); err != nil{
            return nil,err
        }
        slen := binary.BigEndian.Uint16(headerbuf[0:2])
        vlen := binary.BigEndian.Uint64(headerbuf[2:6])

        skeybuf := make([]byte, slen)
        if _,err := io.ReadFull(r,skeybuf); err != nil{
            return nil,err
        }
        req.Skey = string(skeybuf)

        vbuf := make([]byte, vlen)
        if _,err:= io.ReadFull(r, vbuf); err != nil{
            return nil,err
        }
        req.Value = vbuf

    case Fetchflag:
        headerbuf := make([]byte, 2)
        if _,err := io.ReadFull(r, headerbuf); err != nil{
            return nil,err
        }
        flen := binary.BigEndian.Uint16(headerbuf[0:2])

        fbuf := make([]byte, flen)
        if _,err := io.ReadFull(r,fbuf); err!= nil{
            return nil,err
        }
        req.Fkey = string(fbuf)

    case Swapflag:
        headerbuf := make([]byte, 8)
        if _,err := io.ReadFull(r, headerbuf); err != nil{
            return nil,err
        }
        flen := binary.BigEndian.Uint16(headerbuf[0:2])
        slen := binary.BigEndian.Uint16(headerbuf[2:4])
        vlen := binary.BigEndian.Uint32(headerbuf[4:8])

        fbuf := make([]byte, flen)
        if _,err := io.ReadFull(r, fbuf); err!= nil{
            return nil,err
        }
        req.Fkey = string(fbuf)

        sbuf := make([]byte, slen)
        if _,err := io.ReadFull(r, sbuf); err!= nil{
            return nil,err
        }
        req.Skey = string(sbuf)
        
        vbuf := make([]byte, vlen)
        if _,err := io.ReadFull(r, vbuf); err!= nil{
            return nil,err
        }
        req.Value = vbuf
    
    default: 
        return nil,errors.New("request error: request flag not found")
    }
    return req,nil
}