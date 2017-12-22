package mp4

//MP4D TAG TYPE
const (
	MP4ESDescrTag          = 0x03
	MP4DecConfigDescrTag   = 0x04
	MP4DecSpecificDescrTag = 0x05
)

//CodecID
const (
	CodecIDMOV_TEXT          = 0x08
	CodecIDMPEG4             = 0x20
	CodecIDH264              = 0x21
	CodecIDAAC               = 0x40
	CodecIDMP4ALS            = 0x40 /* 14496-3 ALS */
	CodecIDMpeg2VideoMAIN    = 0x61 /* MPEG2 Main */
	CodecIDMpeg2VideoSIMPLE  = 0x60 /* MPEG2 Simple */
	CodecIDMpeg2VideoSNR     = 0x62 /* MPEG2 SNR */
	CodecIDMpeg2VideoSPATIAL = 0x63 /* MPEG2 Spatial */
	CodecIDMpeg2VideoHIGH    = 0x64 /* MPEG2 High */
	CodecIDMPEG2VIDEO        = 0x65 /* MPEG2 422 */
	CodecIDAACMAIN           = 0x66 /* MPEG2 AAC Main */
	CodecIDAACLC             = 0x67 /* MPEG2 AAC Low */
	CodecIDAACSSR            = 0x68 /* MPEG2 AAC SSR */
	CodecIDMp3MPEG2          = 0x69 /* 13818-3 */
	CodecIDMP2               = 0x69 /* 11172-3 */
	CodecIDMPEG1VIDEO        = 0x6A /* 11172-2 */
	CodecIDMp3MPEG1          = 0x6B /* 11172-3 */
	CodecIDMJPEG             = 0x6C /* 10918-1 */
	CodecIDPNG               = 0x6D
	CodecIDJPEG2000          = 0x6E /* 15444-1 */
	CodecIDVC1               = 0xA3
	CodecIDDIRAC             = 0xA4
	CodecIDAC3               = 0xA5
	CodecIDDTS               = 0xA9 /* mp4ra.org */
	CodecIDVORBIS            = 0xDD /* non standard= gpac uses it */
	CodecIDDVD_SUBTITLE      = 0xE0 /* non standard= see unsupported-embedded-subs-2.mp4 */
	CodecIDQCELP             = 0xE1
	CodecIDMPEG4SYSTEMS1     = 0x01
	CodecIDMPEG4SYSTEMS2     = 0x02
	CodecIDNONE              = 0
)
