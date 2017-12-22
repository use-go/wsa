package flv

//Flv data Type
const (
	FlvTagAudio      = 8
	FlvTagVideo      = 9
	FlvTagScriptData = 18
)

// SoundFormat Type
const (
	SoundFormatLinearPCMplatformEndian = 0
	SoundFormatADPCM                   = 1
	SoundFormatMP3                     = 2
	SoundFormatLinearPCMlittleEndian   = 3
	SoundFormatNellymoser16KHzMono     = 4
	SoundFormatNellymoser8KHzMono      = 5
	SoundFormatNellymoser              = 6
	SoundFormatG711ALawPCM             = 7
	SoundFormatG711muLawPCM            = 8
	SoundFormatreserved                = 9
	SoundFormatAAC                     = 10
	SoundFormatSpeex                   = 11
	SoundFormatMP3_8KHz                = 14
	SoundFormatDeviceSpecificSound     = 15
)

//Sound Rate
const (
	SoundRate5_5K = 0
	SoundRate11K  = 1
	SoundRate22K  = 2
	SoundRate44K  = 3
)

//Sound Size
const (
	SoundSize8Bit  = 0
	SoundSize16Bit = 1
)

//Sound Type
const (
	SndMono   = 0
	SndStereo = 1
)

//AAC type
const (
	AACSequenceHeader = 0
	AACRaw            = 1
)

//video 	FrameType
const (
	FrameTypeKeyframe             = 1
	FrameTypeInterFrame           = 2
	FrameTypeDisposableInterFrame = 3 //H263 only
	FrameTypeGeneratedKeyframe    = 4 //server user only
	FrameTypevideoInfoCmdFrame    = 5
)

// Codec Type
const (
	CodecIDJPEG               = 1
	CodecIDSorenSonH263       = 2
	CodecIDScreenVideo        = 3
	CodecIDOn2VP6             = 4
	CodecIDOn2Vp6AlphaChannel = 5
	CodecIDScreenVideoV2      = 6
	CodecIDAVC                = 7
)

//AVC Type
const (
	AVCHeader = 0
	AVCNALU   = 1
)

type FlvTag struct {
	TagType   uint8
	Timestamp uint32
	StreamID  uint32
	Data      []byte
}

type AudioTag struct {
}

type VideoTag struct {
}

func GetAudioTag(flvTag *FlvTag) (result *AudioTag, err error) {
	return
}

func GetVideoTag(flvTag *FlvTag) (result *VideoTag, err error) {
	return
}

//Copy Tag Type
func (flvTag *FlvTag) Copy() (dst *FlvTag) {
	dst = &FlvTag{}
	dst.StreamID = flvTag.StreamID
	dst.TagType = flvTag.TagType
	dst.Timestamp = flvTag.Timestamp
	if len(flvTag.Data) > 0 {
		dst.Data = make([]byte, len(flvTag.Data))
		copy(dst.Data, flvTag.Data)
	}
	return
}
