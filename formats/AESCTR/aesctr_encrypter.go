package aesctr

import (
	"crypto/aes"
	"fmt"
	"io"
)

// AESCTR is a stream wrapper that allows either encryption or decrytion around a stream
// In decryption mode, it wraps a reader and exposes a new reader that will transparently read from the wrapped stream, and perform the decode on it

//As we need to seek around in the stream, the golang default doesnt work for us

type AESCTREncrypter struct {
	sourceStream io.Reader
	counter      *ctr

	backbuffer []byte
}

func NewAESCTREncrypter(reader io.Reader, key, prefix, postfix []byte) (*AESCTREncrypter, error) {
	aesBlockCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return &AESCTREncrypter{
		sourceStream: reader,
		backbuffer:   []byte{},
		counter:      Newctr(aesBlockCipher, aesBlockCipher.BlockSize(), prefix, postfix),
	}, nil
}

func (A *AESCTREncrypter) Seek(i uint64) {
	A.counter.Seek(i)
}
func (A *AESCTREncrypter) Read(p []byte) (int, error) {

	read, err := A.sourceStream.Read(p)
	if read == 0 && err != nil {
		return read, fmt.Errorf("AES CTR Decrypter: read - %w", err)
	}
	// If we read less than the block size, read out more to finish the section
	// This is probably not normally wise, but its padded on the other side

	A.counter.XORKeyStream(p[0:read])

	return read, err

}
