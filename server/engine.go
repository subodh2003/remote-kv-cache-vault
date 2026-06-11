package main

import (
	"errors"
	"sync"
)

const Bucketcnt = 256

type Bucket struct {
	mu    sync.RWMutex
	items map[uint32][]byte
}

type Vault struct {
	buckets [Bucketcnt]*Bucket
}

func NewVault() *Vault {
	v := &Vault{}
	for i := 0; i < Bucketcnt; i++ {
		v.buckets[i] = &Bucket{
			items: make(map[uint32][]byte),
		}
	}
	dummyValue := make([]byte, 1024*1024)
	for i := 0 ; i < 1000; i++ {
		randomKey := uint32(i) 
		idx := v.getBucketIndex(randomKey)
				v.buckets[idx].items[randomKey] = dummyValue
	}
	return v
}

func (v *Vault) getBucketIndex(key uint32) int {
	return int(key % Bucketcnt)
}

func (v *Vault) Store(key uint32, value []byte) {
	idx := v.getBucketIndex(key)
	bucket := v.buckets[idx]
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	bucket.items[key] = value
}

func (v *Vault) Fetch(key uint32) ([]byte, error) {
	idx := v.getBucketIndex(key)
	bucket := v.buckets[idx]
	bucket.mu.RLock()
	defer bucket.mu.RUnlock() // Corrected to RUnlock()

	value, fstatus := bucket.items[key]
	if !fstatus {
		return nil, errors.New("error: block cache miss")
	}
	return value, nil
}

func (v *Vault) Swap(fkey uint32, skey uint32, svalue []byte) ([]byte, error) {
	fidx := v.getBucketIndex(fkey)
	sidx := v.getBucketIndex(skey)

	var firstLock, secondLock *sync.RWMutex
	if fidx < sidx {
		firstLock = &v.buckets[fidx].mu
		secondLock = &v.buckets[sidx].mu
	} else {
		firstLock = &v.buckets[sidx].mu
		secondLock = &v.buckets[fidx].mu
	}

	if fidx == sidx {
		firstLock.Lock() 
		defer firstLock.Unlock()

		bucket := v.buckets[fidx]
		fvalue, fstatus := bucket.items[fkey]
		if !fstatus {
			return nil, errors.New("swap error: fetch key not found in vault")
		}

		bucket.items[skey] = svalue
		if fkey != skey {
			delete(bucket.items, fkey)
		}
		return fvalue, nil
	}

	firstLock.Lock()
	secondLock.Lock()
	defer firstLock.Unlock()
	defer secondLock.Unlock()

	fvalue, fstatus := v.buckets[fidx].items[fkey]
	if !fstatus {
		return nil, errors.New("swap error: fetch key not found in vault")
	}

	v.buckets[sidx].items[skey] = svalue
	delete(v.buckets[fidx].items, fkey)

	return fvalue, nil
}
