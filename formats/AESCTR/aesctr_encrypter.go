package aesctr

import (
	"crypto/aes"
	"errors"
	"fmt"
	"io"
)

// AESCTR is a stream wrapper that allows either encryption or decrytion around a stream
// In decryption mode, it wraps a reader and exposes a new reader that will transparently read from the wrapped stream, and perform the decode on it

//As we need to seek around in the stream, the golang default doesnt work for us

type AESCTREncrypter struct {
	sourceStream io.Reader
	counter      *ctr
}

func NewAESCTREncrypter(reader io.Reader, key, prefix, postfix []byte) (*AESCTREncrypter, error) {
	aesBlockCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return &AESCTREncrypter{
		sourceStream: reader,
		counter:      Newctr(aesBlockCipher, aesBlockCipher.BlockSize(), prefix, postfix),
	}, nil
}

func (A *AESCTREncrypter) Seek(i uint64) {
	A.counter.Seek(i)
}
func (A *AESCTREncrypter) Read(p []byte) (int, error) {
	//Read up to N bytes out, block aligned
	blockSize := A.counter.Blocksize
	n := len(p)
	if n%blockSize != 0 {
		n -= (n % blockSize)
	}
	read, err := A.sourceStream.Read(p)
	if read == 0 {
		return read, fmt.Errorf("AES CTR Decrypter: read - %w", err)
	}
	// One issue to handle when time permits is read != n, and its no longger block aligned
	if read != n {
		if read%blockSize != 0 {
			return 0, errors.New("unhandled, read not of aes block size")
		}
		n = read
	}
	A.counter.XORKeyStream(p[0:n])

	return n, err

}
