package flv

import (
	"os"

	"github.com/use-go/websocket-streamserver/logger"
)

type FlvFileReader struct {
	fp *os.File
}

func (flvfileReader *FlvFileReader) Init(name string) error {
	var err error
	flvfileReader.fp, err = os.Open(name)
	if err != nil {
		logger.LOGE(err.Error())
		return err
	}
	tmp := make([]byte, 13)
	_, err = flvfileReader.fp.Read(tmp)
	return err
}

func (flvfileReader *FlvFileReader) GetNextTag() (tag *FlvTag, err error) {
	buf := make([]byte, 11)
	_, err = flvfileReader.fp.Read(buf)
	if err != nil {
		return
	}
	tag = &FlvTag{}
	tag.TagType = buf[0]
	dataSize := (int(int(buf[1])<<16) | (int(buf[2]) << 8) | (int(buf[3])))
	tag.Timestamp = uint32(int(int(buf[7])<<24) | (int(buf[4]) << 16) | (int(buf[5]) << 8) | (int(buf[6])))

	tag.Data = make([]byte, dataSize)
	_, err = flvfileReader.fp.Read(tag.Data)
	if err != nil {
		return
	}
	buf = make([]byte, 4)
	flvfileReader.fp.Read(buf)
	return
}

func (flvfileReader *FlvFileReader) Close() {
	if flvfileReader.fp != nil {
		flvfileReader.fp.Close()
	}
}
