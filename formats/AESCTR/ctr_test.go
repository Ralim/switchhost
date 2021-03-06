package aesctr_test

import (
	"bytes"
	"testing"

	aesctr "github.com/ralim/switchhost/formats/AESCTR"
)

func TestNSZ(t *testing.T) {
	key := []byte{0x8F, 0x60, 0x31, 0xC3, 0x15, 0x5D, 0x2B, 0x11, 0x6A, 0x30, 0x31, 0x32, 0x33, 0x34, 0x15, 0x40}
	iv := []byte{0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	payloadIn := []byte{0x82, 0xF4, 0x74, 0x3B, 0xC1, 0x90, 0xB4, 0x4E, 0x0D, 0x6F, 0xCC, 0x53, 0x3E, 0xD6, 0xD2, 0x89, 0xA1, 0x24, 0x23, 0x50, 0xA1, 0x13, 0x0A, 0xC9, 0xA4, 0x37, 0x76, 0xEC, 0x26, 0x7B, 0xE8, 0xAA}
	i := 163884
	expectedOutput := []byte{18, 179, 69, 115, 253, 140, 59, 165, 113, 229, 46, 67, 46, 77, 227, 65, 217, 223, 61, 154, 141, 198, 155, 246, 246, 52, 68, 75, 36, 187, 47, 124}

	reader := bytes.NewReader(payloadIn)
	crypter, err := aesctr.NewAESCTREncrypter(reader, key, iv, []byte{})
	if err != nil {
		t.Error(err)
	}
	crypter.Seek(uint64(i))
	output := make([]byte, len(expectedOutput))
	_, err = crypter.Read(output)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(output, expectedOutput) {
		t.Error("Mismatched outputs", output, expectedOutput)
	}

}
