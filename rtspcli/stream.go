package rtspcli

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/format/rtsp/sdp"
)

//Stream info
type Stream struct {
	av.CodecData
	Sdp sdp.Media

	// h264
	fuBuffer []byte
	sps      []byte
	pps      []byte

	gotpkt    bool
	pkt       av.Packet
	timestamp uint32
}

//IsAudio Check Audio
func (strem Stream) IsAudio() bool {
	return strem.Sdp.AVType == "audio"
}

//IsVideo Check Video
func (strem Stream) IsVideo() bool {
	return strem.Sdp.AVType == "video"
}

func (strem *Stream) handleH264Payload(naluType byte, timestamp uint32, packet []byte) (err error) {
	/*
		Table 7-1 – NAL unit type codes
		1   ￼Coded slice of a non-IDR picture
		5    Coded slice of an IDR picture
		6    Supplemental enhancement information (SEI)
		7    Sequence parameter set
		8    Picture parameter set
	*/
	switch naluType {
	case 7, 8:
		// sps/pps

	default:
		if naluType == 5 {
			strem.pkt.IsKeyFrame = true
		}
		strem.gotpkt = true
		strem.pkt.Data = packet
		strem.timestamp = timestamp
	}

	return
}

func (strem *Stream) handlePacket(timestamp uint32, packet []byte) (err error) {
	switch strem.Type() {
	case av.H264:
		/*
			+---------------+
			|0|1|2|3|4|5|6|7|
			+-+-+-+-+-+-+-+-+
			|F|NRI|  Type   |
			+---------------+
		*/
		naluType := packet[0] & 0x1f

		/*
			NAL Unit  Packet    Packet Type Name               Section
			Type      Type
			-------------------------------------------------------------
			0        reserved                                     -
			1-23     NAL unit  Single NAL unit packet             5.6
			24       STAP-A    Single-time aggregation packet     5.7.1
			25       STAP-B    Single-time aggregation packet     5.7.1
			26       MTAP16    Multi-time aggregation packet      5.7.2
			27       MTAP24    Multi-time aggregation packet      5.7.2
			28       FU-A      Fragmentation unit                 5.8
			29       FU-B      Fragmentation unit                 5.8
			30-31    reserved                                     -
		*/

		switch {
		case naluType >= 1 && naluType <= 23:
			if err = strem.handleH264Payload(naluType, timestamp, packet); err != nil {
				return
			}

		case naluType == 28: // FU-A
			/*
				0                   1                   2                   3
				0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
				+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
				| FU indicator  |   FU header   |                               |
				+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+                               |
				|                                                               |
				|                         FU payload                            |
				|                                                               |
				|                               +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
				|                               :...OPTIONAL RTP padding        |
				+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
				Figure 14.  RTP payload format for FU-A
				The FU indicator octet has the following format:
				+---------------+
				|0|1|2|3|4|5|6|7|
				+-+-+-+-+-+-+-+-+
				|F|NRI|  Type   |
				+---------------+
				The FU header has the following format:
				+---------------+
				|0|1|2|3|4|5|6|7|
				+-+-+-+-+-+-+-+-+
				|S|E|R|  Type   |
				+---------------+
				S: 1 bit
				When set to one, the Start bit indicates the start of a fragmented
				NAL unit.  When the following FU payload is not the start of a
				fragmented NAL unit payload, the Start bit is set to zero.
				E: 1 bit
				When set to one, the End bit indicates the end of a fragmented NAL
				unit, i.e., the last byte of the payload is also the last byte of
				the fragmented NAL unit.  When the following FU payload is not the
				last fragment of a fragmented NAL unit, the End bit is set to
				zero.
				R: 1 bit
				The Reserved bit MUST be equal to 0 and MUST be ignored by the
				receiver.
				Type: 5 bits
				The NAL unit payload type as defined in table 7-1 of [1].
			*/
			fuIndicator := packet[0]
			fuHeader := packet[1]
			isStart := fuHeader&0x80 != 0
			isEnd := fuHeader&0x40 != 0
			naluType := fuHeader & 0x1f
			if isStart {
				strem.fuBuffer = []byte{fuIndicator&0xe0 | fuHeader&0x1f}
			}
			strem.fuBuffer = append(strem.fuBuffer, packet[2:]...)
			if isEnd {
				if err = strem.handleH264Payload(naluType, timestamp, strem.fuBuffer); err != nil {
					return
				}
			}

		case naluType == 24:
			err = fmt.Errorf("rtsp: unsupported H264 STAP-A")
			return

		default:
			err = fmt.Errorf("rtsp: unsupported H264 naluType=%d", naluType)
			return
		}

	case av.AAC:
		strem.gotpkt = true
		strem.pkt.Data = packet[4:]
		strem.timestamp = timestamp

	default:
		strem.gotpkt = true
		strem.pkt.Data = packet
		strem.timestamp = timestamp
	}
	return
}

func (strem *SocketChannel) parseBlock(blockNo int, packet []byte) (streamIndex int, err error) {
	if blockNo%2 != 0 {
		// rtcp block
		return
	}

	streamIndex = blockNo / 2
	stream := strem.streams[streamIndex]

	/*
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
	*/

	if len(packet) < 8 {
		err = fmt.Errorf("rtp packet too short")
		return
	}
	payloadOffset := 12 + int(packet[0]&0xf)*4
	if payloadOffset+2 > len(packet) {
		err = fmt.Errorf("rtp packet too short")
		return
	}

	timestamp := binary.BigEndian.Uint32(packet[4:8])
	payload := packet[payloadOffset:]

	/*
		PT 	Encoding Name 	Audio/Video (A/V) 	Clock Rate (Hz) 	Channels 	Reference
		0	PCMU	A	8000	1	[RFC3551]
		1	Reserved
		2	Reserved
		3	GSM	A	8000	1	[RFC3551]
		4	G723	A	8000	1	[Vineet_Kumar][RFC3551]
		5	DVI4	A	8000	1	[RFC3551]
		6	DVI4	A	16000	1	[RFC3551]
		7	LPC	A	8000	1	[RFC3551]
		8	PCMA	A	8000	1	[RFC3551]
		9	G722	A	8000	1	[RFC3551]
		10	L16	A	44100	2	[RFC3551]
		11	L16	A	44100	1	[RFC3551]
		12	QCELP	A	8000	1	[RFC3551]
		13	CN	A	8000	1	[RFC3389]
		14	MPA	A	90000		[RFC3551][RFC2250]
		15	G728	A	8000	1	[RFC3551]
		16	DVI4	A	11025	1	[Joseph_Di_Pol]
		17	DVI4	A	22050	1	[Joseph_Di_Pol]
		18	G729	A	8000	1	[RFC3551]
		19	Reserved	A
		20	Unassigned	A
		21	Unassigned	A
		22	Unassigned	A
		23	Unassigned	A
		24	Unassigned	V
		25	CelB	V	90000		[RFC2029]
		26	JPEG	V	90000		[RFC2435]
		27	Unassigned	V
		28	nv	V	90000		[RFC3551]
		29	Unassigned	V
		30	Unassigned	V
		31	H261	V	90000		[RFC4587]
		32	MPV	V	90000		[RFC2250]
		33	MP2T	AV	90000		[RFC2250]
		34	H263	V	90000		[Chunrong_Zhu]
		35-71	Unassigned	?
		72-76	Reserved for RTCP conflict avoidance				[RFC3551]
		77-95	Unassigned	?
		96-127	dynamic	?			[RFC3551]
	*/
	//payloadType := packet[1]&0x7f

	if strem.DebugConn {
		//fmt.Println("packet:", stream.Type(), "offset", payloadOffset, "pt", payloadType)
		if len(packet) > 24 {
			fmt.Println(hex.Dump(packet[:24]))
		}
	}

	if err = stream.handlePacket(timestamp, payload); err != nil {
		return
	}

	return
}
