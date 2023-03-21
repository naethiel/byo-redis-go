package main

import "time"

type StoreKey string
type StoreValue struct {
	Val        []byte
	CreateDate time.Time
}
type Store map[StoreKey]StoreValue

func (s Store) Set(key, value []byte) {
	k := StoreKey(key)

	s[k] = StoreValue{
		Val:        value,
		CreateDate: time.Now(),
	}
}

func (s Store) Get(key []byte) ([]byte, bool) {
	k := StoreKey(key)

	stored, exists := s[k]

	if !exists {
		return []byte{}, false
	}

	return stored.Val, true
}
