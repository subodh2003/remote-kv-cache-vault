"""Independent Python re-implementation of the binary TCP protocol.

This is deliberately separate from client/encoding.go — that file lives in
Go's `package main` and can't be imported from Python, so the QA layer is a
genuine black-box client talking the wire protocol, not a wrapper around the
existing Go client.

Wire format (verified against server/protocol.go):
  STORE: flag(1)=1  key(4,BE)  vlen(4,BE)  value(vlen)   -> ack: 1 byte (1=ok)
  FETCH: flag(1)=2  key(4,BE)                            -> status:1 (0=miss,1=hit)
                                                              if hit: len(4,BE) value(len)
  SWAP:  flag(1)=0  fkey(4,BE) skey(4,BE) vlen(4,BE) value(vlen)
                                                        -> status:1 (0=miss,1=hit)
                                                              if hit: len(4,BE) old_fkey_value(len)

SWAP semantics (from engine.go, not "exchange two values"):
  - reads fkey's current value (miss -> nothing changes, status 0)
  - writes `value` (from the request) into skey
  - deletes fkey, unless fkey == skey
  - returns the OLD value that was at fkey
"""

import socket
import struct

STORE = 1
FETCH = 2
SWAP = 0


class CacheMiss(Exception):
    """Raised when a FETCH (or later, SWAP) reports a cache miss."""


def recv_exact(sock: socket.socket, n: int) -> bytes:
    """Read exactly n bytes, looping over recv() as needed.

    TCP is a byte stream, not a message protocol — a single recv() is not
    guaranteed to return all n bytes at once, especially once payloads are a
    few KB. Relying on one recv() call is a common bug in ad-hoc protocol
    clients, so this is the one thing every request/response function here
    routes through.
    """
    buf = bytearray()
    while len(buf) < n:
        chunk = sock.recv(n - len(buf))
        if not chunk:
            raise ConnectionError(f"connection closed after {len(buf)}/{n} bytes")
        buf.extend(chunk)
    return bytes(buf)


def store(sock: socket.socket, key: int, value: bytes) -> None:
    header = struct.pack(">BII", STORE, key, len(value))
    sock.sendall(header + value)
    ack = recv_exact(sock, 1)
    if ack[0] != 1:
        raise RuntimeError(f"STORE failed, server returned ack={ack[0]!r}")


def fetch(sock: socket.socket, key: int) -> bytes:
    header = struct.pack(">BI", FETCH, key)
    sock.sendall(header)
    status = recv_exact(sock, 1)
    if status[0] == 0:
        raise CacheMiss(f"key {key} not present")
    (length,) = struct.unpack(">I", recv_exact(sock, 4))
    return recv_exact(sock, length)


def swap(sock: socket.socket, fkey: int, skey: int, value: bytes) -> bytes:
    """Returns the OLD value that was stored at fkey. Raises CacheMiss if
    fkey had nothing stored (in which case skey is left untouched)."""
    header = struct.pack(">BIII", SWAP, fkey, skey, len(value))
    sock.sendall(header + value)
    status = recv_exact(sock, 1)
    if status[0] == 0:
        raise CacheMiss(f"fkey {fkey} not present")
    (length,) = struct.unpack(">I", recv_exact(sock, 4))
    return recv_exact(sock, length)
