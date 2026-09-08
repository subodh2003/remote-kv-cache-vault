"""Test 4: SWAP

SWAP is not a simple exchange — see the docstring in protocol.py. These
tests pin down the actual behavior implemented in engine.go:
  - normal swap: fkey deleted, skey gets the new value, old fkey value returned
  - missing fkey: reported as a miss, nothing is changed
  - fkey == skey: value is overwritten in place, NOT deleted
"""

import pytest

from protocol import CacheMiss, fetch, store, swap


def test_swap_moves_new_value_in_and_deletes_fkey(client_socket, unique_key):
    fkey = unique_key()
    skey = unique_key()
    old_value = b"old-value-at-fkey"
    new_value = b"new-value-for-skey"

    store(client_socket, fkey, old_value)

    returned = swap(client_socket, fkey, skey, new_value)
    assert returned == old_value, "SWAP must return the value fkey held before the swap"

    assert fetch(client_socket, skey) == new_value, "skey should now hold the request's value"

    with pytest.raises(CacheMiss):
        fetch(client_socket, fkey)  # fkey must be deleted after a fkey != skey swap


def test_swap_missing_fkey_reports_cache_miss(client_socket, unique_key):
    fkey = unique_key()       # never STOREd
    skey = unique_key()

    with pytest.raises(CacheMiss):
        swap(client_socket, fkey, skey, b"irrelevant")

    # side effect check: a miss on fkey must not have written anything to skey
    with pytest.raises(CacheMiss):
        fetch(client_socket, skey)


def test_swap_same_key_overwrites_without_deleting(client_socket, unique_key):
    key = unique_key()
    old_value = b"before"
    new_value = b"after"

    store(client_socket, key, old_value)

    returned = swap(client_socket, key, key, new_value)
    assert returned == old_value

    # fkey == skey, so the delete step must not fire — the key should still
    # exist, now holding the new value.
    assert fetch(client_socket, key) == new_valuegit add qafiles/
git commit -m "Add pytest QA automation layer: STORE/FETCH/SWAP validation over TCP"
git push