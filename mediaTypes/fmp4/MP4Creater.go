package fmp4

import (
	"github.com/use-go/websocket-streamserver/mediaTypes/flv"
)

func ftypBox() (box *Box) {
	box = &Box{}
	box.Init([]byte("ftyp"))
	box.PushBytes([]byte("iso5"))
	box.Push4Bytes(1)
	box.PushBytes([]byte("iso5"))
	box.PushBytes([]byte("dash"))
	return box
}

func moovBox(audioTag, vdeoTag *flv.FlvTag) (box *Box) {
	box = &Box{}
	box.Init([]byte("moov"))
	//mvhd

	//mvex
	//tracks
	return
}

func mvhdBox() (box *Box) {
	return
}

func mvexBox(audioTag, vdeoTag *flv.FlvTag) (box *Box) {
	return
}

func trexBox(tag *flv.FlvTag) (box *Box) {
	return
}

func trakBox(tag *flv.FlvTag) (box *Box) {
	return
}
