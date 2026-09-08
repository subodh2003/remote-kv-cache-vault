"""Test 1: server availability
Test 2: STORE -> FETCH round trip
Test 3: FETCH on a genuinely missing key
"""

import pytest

from protocol import CacheMiss, fetch, store


def test_server_accepts_connections(server):
    """The `server` fixture already asserts readiness by polling for a TCP
    connect before yielding — if that failed, this test never runs. Kept as
    its own test so a broken build/bind shows up as one clear failure
    instead of every other test failing with a confusing connection error."""
    host, port = server
    assert host and port


def test_store_then_fetch_returns_same_value(client_socket, unique_key):
    """The core functional guarantee of the system: whatever you STORE is
    exactly what a later FETCH returns. Exercises the full path once:
    Python -> TCP -> Go server -> Vault -> response -> Python."""
    key = unique_key()
    value = b"hello-from-pytest"
    store(client_socket, key, value)
    result = fetch(client_socket, key)
    assert result == value


def test_fetch_missing_key_reports_cache_miss(client_socket, unique_key):
    """unique_key() is >=10000 and never STOREd by this test, so this must
    be a genuine miss — not the pre-seeded dummy 1KB blocks the vault ships
    with for keys 0..9999."""
    key = unique_key()
    with pytest.raises(CacheMiss):
        fetch(client_socket, key)