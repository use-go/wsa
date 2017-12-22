package mp4

import (
	"bytes"
	"container/list"
	"encoding/binary"
	"log"
)

//MP4采用盒子套盒子的递归方式存储数据
type boxOffset struct {
	longBox bool //是否是长的盒子，长盒子用8个字节表示长度，否则4个
	pos     int  //写入长度的位置
}

type MP4Box struct {
	writer bytes.Buffer
	offset list.List
}

//压入一个盒子
func (mp4Box *MP4Box) Push(name []byte) {
	offset := &boxOffset{}
	offset.longBox = false
	offset.pos = mp4Box.writer.Len()
	mp4Box.offset.PushBack(offset)
	mp4Box.Push4Bytes(0)
	mp4Box.PushBytes(name)
}

//压入一个长盒子
func (mp4Box *MP4Box) PushLongBox(name []byte) {
	log.Fatal("do not call mp4Box func ,bytes.Buffer 32bit only")
	offset := &boxOffset{}
	offset.longBox = true
	offset.pos = mp4Box.writer.Len()
	mp4Box.offset.PushBack(offset)
	mp4Box.Push4Bytes(1)
	mp4Box.Push8Bytes(0)
	mp4Box.PushBytes(name)
}

//弹出一个盒子
func (mp4Box *MP4Box) Pop() {
	if mp4Box.offset.Len() == 0 {
		return
	}
	offset := mp4Box.offset.Back().Value.(*boxOffset)
	boxSize := mp4Box.writer.Len() - offset.pos
	if offset.longBox == true {
		data := mp4Box.writer.Bytes()
		data[offset.pos+0] = 0
		data[offset.pos+1] = 0
		data[offset.pos+2] = 0
		data[offset.pos+3] = 1
		//这里不应该使用这个，因为bytes.buffer 只有32位
		log.Fatal("can not pop longbox mp4Box way,bytes.buffer 32bit only")
	} else {
		data := mp4Box.writer.Bytes()
		data[offset.pos+0] = byte((boxSize >> 24) & 0xff)
		data[offset.pos+1] = byte((boxSize >> 16) & 0xff)
		data[offset.pos+2] = byte((boxSize >> 8) & 0xff)
		data[offset.pos+3] = byte((boxSize >> 0) & 0xff)
	}

	mp4Box.offset.Remove(mp4Box.offset.Back())
}

//清空整个盒子
func (mp4Box *MP4Box) Flush() []byte {
	if mp4Box.writer.Len() == 0 {
		return nil
	}
	data := make([]byte, mp4Box.writer.Len())
	copy(data, mp4Box.writer.Bytes())
	mp4Box.writer.Reset()
	return data
}

func (mp4Box *MP4Box) Push8Bytes(data uint64) {
	err := binary.Write(&mp4Box.writer, binary.BigEndian, data)
	if err != nil {
		log.Println(err.Error())
	}
}

func (mp4Box *MP4Box) Push4Bytes(data uint32) {
	err := binary.Write(&mp4Box.writer, binary.BigEndian, data)
	if err != nil {
		log.Println(err.Error())
	}
}

func (mp4Box *MP4Box) Push2Bytes(data uint16) {
	err := binary.Write(&mp4Box.writer, binary.BigEndian, data)
	if err != nil {
		log.Println(err.Error())
	}
}

func (mp4Box *MP4Box) PushByte(data uint8) {
	err := mp4Box.writer.WriteByte(data)
	if err != nil {
		log.Println(err.Error())
	}
}

func (mp4Box *MP4Box) PushBytes(data []byte) {
	_, err := mp4Box.writer.Write(data)
	if err != nil {
		log.Println(err.Error())
	}
}
