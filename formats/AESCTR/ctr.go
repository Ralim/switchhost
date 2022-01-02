package aesctr

import (
	"crypto/cipher"
	"encoding/binary"
)

const streamBufferSize = 512

type ctr struct {
	Blocksize int

	blockCipher cipher.Block

	counter uint64
	prefix  []byte
	postfix []byte

	out     []byte
	outUsed int
}

func Newctr(cipher cipher.Block, blocksize int, prefix, postfix []byte) *ctr {
	bufSize := streamBufferSize
	if bufSize < blocksize {
		bufSize = blocksize
	}
	return &ctr{
		counter:     0,
		prefix:      prefix[0:8],
		postfix:     postfix,
		out:         make([]byte, 0, bufSize),
		outUsed:     0,
		Blocksize:   blocksize,
		blockCipher: cipher,
	}
}

func (c *ctr) Seek(i uint64) {
	c.counter = i >> 4
	//Clear the buffer
	c.outUsed = len(c.out)
	c.refill() // Re-gens the crypto buffer
}

func (A *ctr) getKey() []byte {
	// +------+--------------+-------+
	// |prefix| counter value|postfix|
	// +------+--------------+-------+
	output := A.prefix
	counter := make([]byte, 8)
	binary.BigEndian.PutUint64(counter, A.counter)
	output = append(output, counter...)
	output = append(output, A.postfix...)
	return output
}

func (A *ctr) refill() {
	remain := len(A.out) - A.outUsed
	copy(A.out, A.out[A.outUsed:])
	A.out = A.out[:cap(A.out)]
	bs := A.blockCipher.BlockSize()
	for remain <= len(A.out)-bs {
		A.blockCipher.Encrypt(A.out[remain:], A.getKey())
		remain += bs
		A.counter++
	}
	A.out = A.out[:remain]
	A.outUsed = 0
}

func (A *ctr) XORKeyStream(dst []byte) {
	for len(dst) > 0 {
		if A.outUsed >= len(A.out)-A.blockCipher.BlockSize() {
			A.refill()
		}
		// Want to use xorBytes from inside crypto but its not exported
		// so for now, this is the best we have
		// n := xorBytes(dst, dst, A.out[A.outUsed:])

		n := len(A.out) - A.outUsed
		if len(dst) < n {
			n = len(dst)
		}

		for i := 0; i < n; i++ {
			dst[i] = dst[i] ^ A.out[A.outUsed+i]
		}

		dst = dst[n:]
		A.outUsed += n
	}
}
