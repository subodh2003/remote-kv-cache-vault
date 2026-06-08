package main

import (
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
