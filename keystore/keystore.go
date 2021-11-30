package keystore

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Keystore is minimal holders for the keys db

type Keystore struct {
	keys map[string]string
}

// NewKeystore creates a new keystore instance from the data in the provided reader
func NewKeystore(r io.Reader) (*Keystore, error) {
	//Reads all lines from the keys file and extracts the ones we care about
	store := &Keystore{
		keys: make(map[string]string),
	}
	if r == nil {
		return store, errors.New("cant load keys from nil reader")
	}
	scanner := bufio.NewScanner(r)
	// Could we use a library to scan this.. yes
	// Should we? :shrug: its a fairly simple file really
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "=")
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			//We only care about lines that start with `key_area_key_application_` or `header_key`
			if key == "header_key" || strings.HasPrefix(key, "key_area_key_application_") {
				store.keys[key] = value
			}
		}
	}

	if len(store.keys) == 0 {
		return store, errors.New("no keys were loaded from the provided database")
	}
	return store, nil

}

func (key *Keystore) GetHeaderKey() ([]byte, error) {
	return key.getKey("header_key")
}
func (key *Keystore) GetAppKey(revision uint8) ([]byte, error) {
	keyName := fmt.Sprintf("key_area_key_application_%02x", revision)
	return key.getKey(keyName)
}

func (key *Keystore) getKey(keyName string) ([]byte, error) {
	KeyString, ok := key.keys[keyName]
	if !ok {
		return []byte{}, fmt.Errorf("key not found - %s", keyName)
	}
	keyBytes, err := hex.DecodeString(KeyString)
	if err != nil {
		return []byte{}, fmt.Errorf("invalid key parse - %w", err)
	}
	return keyBytes, nil
}
