package main

import (
	"errors"
	"hash/fnv"
	"sync"
)

const Bucketcnt = 256

type Bucket struct {
	mu    sync.RWMutex
	items map[string][]byte
}

type Vault struct {
	buckets [Bucketcnt]*Bucket
}

func NewVault() *Vault {
	v := &Vault{}
	for i := 0; i < Bucketcnt; i++ {
		v.buckets[i] = &Bucket{
			items: make(map[string][]byte),
		}
	}
	return v
}

func (v *Vault) getBucketIndex(key string) int {
	hasher := fnv.New32a()
	hasher.Write([]byte(key))
	return int(hasher.Sum32() % Bucketcnt)
}

func (v *Vault) Store(key string, value []byte){
	idx:= v.getBucketIndex(key)
	bucket := v.buckets[idx]
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	bucket.items[key] = value
}

func (v *Vault) Fetch(key string) ([]byte,error){
	idx := v.getBucketIndex(key)
	bucket:= v.buckets[idx]
	bucket.mu.RLock()
	defer bucket.mu.Unlock()

	value,fstatus:= bucket.items[key]
	if !fstatus{
		return nil,errors.New("error: block cache miss")
	}
	return value,nil
}

func (v *Vault) Swap(fkey string, skey string, svalue []byte) ([]byte, error){
	fidx := v.getBucketIndex(fkey)
	sidx := v.getBucketIndex(skey)

	if fidx == sidx {
		bucket := v.buckets[fidx]
		bucket.mu.Lock()
		defer bucket.mu.Unlock()

		fvalue, fstatus:= bucket.items[fkey]
		bucket.items[skey] = svalue
		if !fstatus{
			return nil,errors.New("swap error: fetch key not found in vault")
		}
		return fvalue,nil
	}

	if fidx < sidx{
		v.buckets[fidx].mu.Lock()
		v.buckets[sidx].mu.Lock()
	} else{
		v.buckets[sidx].mu.Lock()
		v.buckets[fidx].mu.Lock()
	}
	defer v.buckets[fidx].mu.Unlock()
	defer v.buckets[sidx].mu.Unlock()

	fvalue , fstatus:= v.buckets[fidx].items[fkey]
	v.buckets[sidx].items[skey] = svalue
	if !fstatus {
		return nil, errors.New("swap error: fetch key not found in vault")
	} 
	return fvalue,nil
}

