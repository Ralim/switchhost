package keystore

import (
	"reflect"
	"strings"
	"testing"
)

func TestNewKeystoreGood(t *testing.T) {
	t.Parallel()
	// Naturally all testing data is FAKE dont even bother trying to use these keys
	var validFile = `
key_area_key_application_00       = 00key
key_area_key_application_ff       = ffkey
header_key                        = 012345678901234567890123456789aa`
	reader := strings.NewReader(validFile)
	result, err := NewKeystore(reader)
	if err != nil {
		t.Error(err)
	}
	if len(result.keys) != 3 {
		t.Errorf("got %d keys, wanted 3 keys", len(result.keys))
	}
	if v, ok := result.keys["key_area_key_application_00"]; !ok || v != "00key" {
		t.Errorf("Got wrong key 0, wanted 00key, given >%s<", v)
	}
	if v, ok := result.keys["key_area_key_application_ff"]; !ok || v != "ffkey" {
		t.Errorf("Got wrong key ff, wanted ffkey, given >%s<", v)
	}
	if v, ok := result.keys["header_key"]; !ok || v != "012345678901234567890123456789aa" {
		t.Errorf("Got wrong key header, wanted 012345678901234567890123456789aa, given >%s<", v)
	}
}

func TestNewKeystoreempty(t *testing.T) {
	t.Parallel()
	// Naturally all testing data is FAKE dont even bother trying to use these keys
	var validFile = "   "
	reader := strings.NewReader(validFile)
	result, err := NewKeystore(reader)
	if err == nil {
		t.Error("should raise error on empty file")
	}
	if len(result.keys) != 0 {
		t.Errorf("got %d keys, wanted 0 keys", len(result.keys))
	}
}

func TestNewKeystoreBadInput(t *testing.T) {
	t.Parallel()
	// Naturally all testing data is FAKE dont even bother trying to use these keys
	result, err := NewKeystore(nil)
	if err == nil {
		t.Error("Should raise error on invalid input")
	}
	if len(result.keys) != 0 {
		t.Errorf("got %d keys, wanted 0 keys", len(result.keys))
	}

}

/******** Fetching ****/

func TestGetHeaderKey(t *testing.T) {
	t.Parallel()
	// Naturally all testing data is FAKE dont even bother trying to use these keys
	var validFile = `
key_area_key_application_00       = 00key
key_area_key_application_ff       = ffkey
header_key                        = 012345678901234567890123456789aa`
	reader := strings.NewReader(validFile)
	keystore, err := NewKeystore(reader)
	if err != nil {
		t.Error(err)
	}
	key, err := keystore.GetHeaderKey()
	if err != nil {
		t.Errorf("should fetch valid key ok - %v", err)
	}
	if !reflect.DeepEqual(key, []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0x01, 0x23, 0x45, 0x67, 0x89, 0x01, 0x23, 0x45, 0x67, 0x89, 0xaa}) {
		t.Errorf("%v is not the expected header key", key)
	}
}

func TestGetAppKey(t *testing.T) {
	t.Parallel()
	// Naturally all testing data is FAKE dont even bother trying to use these keys
	var validFile = `
key_area_key_application_00       = 0099
key_area_key_application_ff       = ffAA
header_key                        = 012345678901234567890123456789aa`
	reader := strings.NewReader(validFile)
	keystore, err := NewKeystore(reader)
	if err != nil {
		t.Error(err)
	}
	key, err := keystore.GetAppKey(0)
	if err != nil {
		t.Errorf("should fetch valid key ok - %v", err)
	}
	if !reflect.DeepEqual(key, []byte{0x00, 0x99}) {
		t.Errorf("%v is not the expected header key", key)
	}
	key, err = keystore.GetAppKey(0xFF)
	if err != nil {
		t.Errorf("should fetch valid key ok - %v", err)
	}
	if !reflect.DeepEqual(key, []byte{0xFF, 0xaa}) {
		t.Errorf("%v is not the expected header key", key)
	}
}

func TestGetKey(t *testing.T) {
	t.Parallel()
	// Naturally all testing data is FAKE dont even bother trying to use these keys
	var validFile = `
key_area_key_application_00       = 0099
header_key                        = corruptKey
`
	reader := strings.NewReader(validFile)
	keystore, err := NewKeystore(reader)
	if err != nil {
		t.Error(err)
	}
	_, err = keystore.getKey("header_key")
	if err == nil {
		t.Error("should fail to parse bad data")
	} else if err.Error() != "invalid key parse - encoding/hex: invalid byte: U+006F 'o'" {
		t.Errorf("Missing key should have the right err; >%s<", err.Error())
	}
	_, err = keystore.getKey("not_here")
	if err == nil {
		t.Error("should fail to find this key")
	} else if err.Error() != "key not found - not_here" {
		t.Errorf("Missing key should have the right err; >%s<", err.Error())
	}
}
