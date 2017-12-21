package wssAPI

import (
	"bytes"
	"encoding/binary"
)

//BitReader read data by bit
type BitReader struct {
	buf    []byte
	curBit int
}

// Init bit array
func (bitReader *BitReader) Init(data []byte) {
	bitReader.curBit = 0
	bitReader.buf = make([]byte, len(data))
	copy(bitReader.buf, data)
}

// ReadBit from Array
func (bitReader *BitReader) ReadBit() int {
	if bitReader.curBit > (len(bitReader.buf) << 3) {
		return -1
	}
	idx := (bitReader.curBit >> 3)
	offset := bitReader.curBit%8 + 1
	bitReader.curBit++
	return int(bitReader.buf[idx]>>uint(8-offset)) & 0x01
}

//ReadBits from stream
func (bitReader *BitReader) ReadBits(num int) int {
	r := 0
	for i := 0; i < num; i++ {
		r |= (bitReader.ReadBit() << uint(num-i-1))
	}
	return r
}

// Read32Bits for 32bit
func (bitReader *BitReader) Read32Bits() uint32 {
	idx := (bitReader.curBit >> 3)
	var r uint32
	binary.Read(bytes.NewReader(bitReader.buf[idx:]), binary.BigEndian, &r)
	bitReader.curBit += 32

	return r
}

// ReadExponentialGolombCode from stream
func (bitReader *BitReader) ReadExponentialGolombCode() int {
	r := 0
	i := 0
	for bitReader.ReadBit() == 0 && (i < 32) {
		i++
	}
	r = bitReader.ReadBits(i)
	r += (1 << uint(i)) - 1
	return r
}

//ReadSE function
func (bitReader *BitReader) ReadSE() int {
	r := bitReader.ReadExponentialGolombCode()
	if (r & 0x01) != 0 {
		r = (r + 1) / 2
	} else {
		r = -(r / 2)
	}
	return r
}

//CopyBits function
func (bitReader *BitReader) CopyBits(num int) int {
	cur := bitReader.curBit
	r := 0
	for i := 0; i < num; i++ {
		r |= (bitReader.copyBit(cur+i) << uint(num-i-1))
	}
	return r
}

func (bitReader *BitReader) copyBit(cur int) int {
	if cur > (len(bitReader.buf) << 3) {
		return -1
	}
	idx := (cur >> 3)
	offset := cur%8 + 1
	return int(bitReader.buf[idx]>>uint(8-offset)) & 0x01
}

//BitsLeft to get left Bits
func (bitReader *BitReader) BitsLeft() int {
	return (len(bitReader.buf) << 3) - bitReader.curBit
}
