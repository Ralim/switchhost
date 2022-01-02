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
	//Read up to N bytes out, block aligned
	if len(A.backbuffer) > 0 {
		if len(A.backbuffer) > len(p) {
			//Fill p and exit
			copy(p, A.backbuffer)
			A.backbuffer = A.backbuffer[len(p):]
			return len(p), nil
		}

		copy(p, A.backbuffer)
		p = p[len(A.backbuffer):]
		A.backbuffer = []byte{}
	}
	read, err := A.sourceStream.Read(p)
	if read == 0 && err != nil {
		return read, fmt.Errorf("AES CTR Decrypter: read - %w", err)
	}
	n := read
	// If we read less than the block size, read out more to finish the section
	// This is probably not normally wise, but its padded on the other side
	blockSize := A.counter.Blocksize

	if n%blockSize != 0 {
		rounded := (n / blockSize) * blockSize
		A.counter.XORKeyStream(p[0:rounded])
		remainder := n % blockSize

		scratch := make([]byte, blockSize)
		copy(scratch, p[rounded:rounded+remainder])
		read, err := A.sourceStream.Read(scratch[remainder:])
		if read == 0 && err != nil {
			return read, fmt.Errorf("AES CTR Decrypter: read - %w", err)
		}
		A.counter.XORKeyStream(scratch)
		copy(p[rounded:], scratch[:remainder])
		A.backbuffer = scratch[remainder:]
	} else {

		A.counter.XORKeyStream(p[0:n])
	}

	return n, err

}
