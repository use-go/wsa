/*
	The RTP header has the following format:
    0                   1                   2                   3
    0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   |V=2|P|X|  CC   |M|     PT      |       sequence number         |
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   |                           timestamp                           |
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   |           synchronization source (SSRC) identifier            |
   +=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+
   |            contributing source (CSRC) identifiers             |
   |                             ....                              |
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   version (V): 2 bits
      This field identifies the version of RTP.  The version defined by
      this specification is two (2).  (The value 1 is used by the first
      draft version of RTP and the value 0 is used by the protocol
      initially implemented in the "vat" audio tool.)
   padding (P): 1 bit
      If the padding bit is set, the packet contains one or more
      additional padding octets at the end which are not part of the
      payload.  The last octet of the padding contains a count of how
      many padding octets should be ignored, including itself.  Padding
      may be needed by some encryption algorithms with fixed block sizes
      or for carrying several RTP packets in a lower-layer protocol data
      unit.
   extension (X): 1 bit
      If the extension bit is set, the fixed header MUST be followed by
      exactly one header extension, with a format defined in Section
      5.3.1.
*/

package rtspsrv

import (
	//	"logger"
	"github.com/use-go/websocket-streamserver/mediatype/amf"
)

//DataType for RTP
const (
	NalTypeSingleNalMIN = 1
	NalTypeSingleNalMAX = 23
	NalTypeStapA        = 24
	NalTypeStapB        = 25
	NalTypeMtap16       = 26
	NalTypeMtap24       = 27
	NalTypeFuA        = 28
	NalTypeFuB        = 29
	PayloadH264        = 96
	PayloadMPA         = 14 //mp3 freq 90000
	RTPH264Freq       = 90000
	RTPMTU = 1500
)

func createRTPHeader(payloadType, seq, timestamp, ssrc uint32) []byte {
	encoder := &amf.AMF0Encoder{}
	encoder.Init()
	version := byte(2)
	padding := byte(0)
	extension := byte(0)
	marker := byte(0)
	cc := byte(0)

	tmp := (version << 6) | (padding << 5) | (extension << 4) | (cc)
	encoder.AppendByte(tmp)
	tmp = (marker << 7) | byte(payloadType)
	encoder.AppendByte(tmp)
	encoder.EncodeInt16(int16(seq))
	encoder.EncodeInt32(int32(timestamp))
	//	logger.LOGD(timestamp)
	encoder.EncodeInt32(int32(ssrc))

	data, _ := encoder.GetData()
	return data
}

func createRTPHeaderAAC(payloadType, seq, timestamp, ssrc uint32) []byte {
	encoder := &amf.AMF0Encoder{}
	encoder.Init()
	version := byte(2)
	padding := byte(0)
	extension := byte(0)
	marker := byte(1)
	cc := byte(0)

	tmp := (version << 6) | (padding << 5) | (extension << 4) | (cc)
	encoder.AppendByte(tmp)
	tmp = (marker << 7) | byte(payloadType)
	encoder.AppendByte(tmp)
	encoder.EncodeInt16(int16(seq))
	encoder.EncodeInt32(int32(timestamp))
	//	logger.LOGD(timestamp)
	encoder.EncodeInt32(int32(ssrc))

	data, _ := encoder.GetData()
	return data
}
