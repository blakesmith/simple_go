package main

import (
	"errors"
	"fmt"
)

var (
	ErrKeyNotFound = errors.New("Error: key not found")
)

type KVStringStore interface {
	Put(key, value string) error
	Get(key string) (string, error)
}

type MemoryStore struct {
	store map[string]string
}

func (m *MemoryStore) Put(key, value string) error {
	m.store[key] = value

	return nil
}

func (m *MemoryStore) Get(key string) (string, error) {
	val, ok := m.store[key]
	if !ok {
		return "", ErrKeyNotFound
	}

	return val, nil
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		store: make(map[string]string),
	}
}

func storeAndGet(store KVStringStore) {
	if err := store.Put("I love", "Lamp"); err != nil {
		fmt.Println("Cannot write to the store")
	}
	fmt.Println("Stored!")
	val, err := store.Get("I love")
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("The value is: %s\n", val)
}

func main() {
	store := NewMemoryStore()
	storeAndGet(store)
}
