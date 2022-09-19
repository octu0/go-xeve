package xeve

import (
	"bytes"
	"encoding/binary"
	"runtime"
	"sync"
	"sync/atomic"
)

type ReturnCode int

const (
	NoMoreFrames             ReturnCode = 205
	OutNotAvailable                     = 204
	FrameDimensionChanged               = 203
	FrameDelayed                        = 202
	ErrBadCRC                           = 201
	ErrWarnCRCIgnored                   = 200
	Ok                                  = 0
	Err                                 = -1
	ErrInvalidArgument                  = -101
	ErrOutOfMemory                      = -102
	ErrReachedMax                       = -103
	ErrUnsupported                      = -104
	ErrUnexpected                       = -105
	ErrUnsupportedColorSpace            = -201
	ErrMalformedBitstream               = -202
	ErrUnknown                          = -32767
)

func (rc ReturnCode) Error() string {
	switch rc {
	// Succeed
	case Ok:
		return "XEVE_OK"
	case NoMoreFrames:
		return "XEVE_OK_NO_MORE_FRM"
	case OutNotAvailable:
		return "XEVE_OK_OUT_NOT_AVAILABLE"
	case FrameDimensionChanged:
		return "XEVE_OK_DIM_CHANGED"
	case FrameDelayed:
		return "XEVE_OK_FRM_DELAYED"
	case ErrBadCRC:
		return "XEVE_ERR_BAD_CRC"
	case ErrWarnCRCIgnored:
		return "XEVE_WARN_CRC_IGNORED"
	// Failed
	case Err:
		return "XEVE_ERR"
	case ErrInvalidArgument:
		return "XEVE_ERR_INVALID_ARGUMENT"
	case ErrOutOfMemory:
		return "XEVE_ERR_OUT_OF_MEMORY"
	case ErrReachedMax:
		return "XEVE_ERR_REACHED_MAX"
	case ErrUnsupported:
		return "XEVE_ERR_UNSUPPORTED"
	case ErrUnexpected:
		return "XEVE_ERR_UNEXPECTED"
	case ErrUnsupportedColorSpace:
		return "XEVE_ERR_UNSUPPORTED_COLORSPACE"
	case ErrMalformedBitstream:
		return "XEVE_ERR_MALFORMED_BITSTREAM"
	}
	return "XEVE_ERR_UNKNOWN"
}

func Succeed(rc ReturnCode) bool {
	if Ok <= rc {
		return true
	}
	return false
}

func Failed(rc ReturnCode) bool {
	if rc < Ok {
		return true
	}
	return false
}

type ColorFormatType uint8

const (
	ColorFormatUnknown   = 0
	ColorFormatYCbCr400  = 10 // Y onlu
	ColorFormatYCbCr420  = 11 // YCbCr 420
	ColorFormatYCbCr422  = 12 // YCbCr 422 narrow chroma
	ColorFormatYCbCr444  = 13 // YCbCr 444
	ColorFormatYCbCr422N = ColorFormatYCbCr422
	ColorFormatYCbCr422W = 18 // YCbCr 422 wide chroma
)

type ConfigType uint16

const (
	ConfigSetForceOut        ConfigType = 102
	ConfigSetFIntra                     = 200
	ConfigSetQP                         = 201
	ConfigSetBPS                        = 202
	ConfigSetVBVSize                    = 203
	ConfigSetFPS                        = 204
	ConfigSetKeyInterval                = 207
	ConfigSetQPMin                      = 208
	ConfigSetQPMax                      = 209
	ConfigSetBUSize                     = 210
	ConfigSetUseDeblock                 = 211
	ConfigSetDeblockAOffset             = 212
	ConfigSetDeblockBOffset             = 213
	ConfigSetSEICMD                     = 300
	ConfigSetUsePicSignature            = 301
	ConfigGetComplexity                 = 500
	ConfigGetSpeed                      = 501
	ConfigGetQPMin                      = 600
	ConfigGetQPMax                      = 601
	ConfigGetQP                         = 602
	ConfigGetRCT                        = 603
	ConfigGetBPS                        = 604
	ConfigGetFPS                        = 605
	ConfigGetKeyInterval                = 608
	ConfigGetBUSize                     = 609
	ConfigGetUseDeblock                 = 610
	ConfigGetClosedGOP                  = 611
	ConfigGetHierarchicalGOP            = 612
	ConfigGetDeblockAOffset             = 613
	ConfigGetDeblockBOffset             = 614
	ConfigGetWidth                      = 701
	ConfigGetHeight                     = 702
	ConfigGetRECON                      = 703
	ConfigGetSupportProfile             = 704
)

type NALUnitType uint8

const (
	NALUnitNonIDR NALUnitType = 0
	NALUnitIDR                = 1
	NALUnitSPS                = 24
	NALUnitPPS                = 25
	NALUnitAPS                = 26
	NALUnitFD                 = 27
	NALUnitSEI                = 28
)

func (n NALUnitType) String() string {
	switch n {
	case NALUnitNonIDR:
		return "NonIDR"
	case NALUnitIDR:
		return "IDR"
	case NALUnitSPS:
		return "SPS"
	case NALUnitPPS:
		return "PPS"
	case NALUnitAPS:
		return "APS"
	case NALUnitFD:
		return "FD"
	case NALUnitSEI:
		return "SEI"
	default:
		return "Unknown"
	}
}

type SliceType int8

const (
	SliceUnknown SliceType = -1
	SliceB                 = 0
	SliceP                 = 1
	SliceI                 = 2
)

func (s SliceType) String() string {
	switch s {
	case SliceB:
		return "B"
	case SliceP:
		return "P"
	case SliceI:
		return "I"
	default:
		return "Unknown"
	}
}

type ProfileType uint8

const (
	ProfileBaseline ProfileType = 0
	ProfileMain                 = 1
)

type PresetType uint8

const (
	PresetDefault PresetType = 0
	PresetFast               = 1
	PresetMedium             = 2
	PresetSlow               = 3
	PresetPlacebo            = 4
)

type TuneType uint8

const (
	TuneNone        TuneType = 0
	TuneZeroLatency          = 1
	TunePSNR                 = 2
)

type RateControlType uint8

const (
	RateControlCQP RateControlType = 0
	RateControlABR                 = 1
	RateControlCRF                 = 2
)

type GOPType uint8

const (
	GOPOpen   GOPType = 0
	GOPClosed         = 1
)

var (
	pool = &sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 0, 1024))
		},
	}
)

func createReleasePoolFunc(buf *bytes.Buffer) func() {
	return func() {
		pool.Put(buf)
	}
}

type NALUnit struct {
	NALUnit   NALUnitType
	Slice     SliceType
	Data      []byte
	closeFunc func()
	closed    int32
}

func (n *NALUnit) HasData() bool {
	if 0 < len(n.Data) {
		return true
	}
	return false
}

func (n *NALUnit) setFinalizer() {
	runtime.SetFinalizer(n, func(me *NALUnit) {
		me.Close()
	})
}

func (n *NALUnit) SplitNAL() []NAL {
	nals := make([]NAL, 0)
	pos := uint32(0)
	dataSize := uint32(len(n.Data))
	for {
		size := binary.BigEndian.Uint32(n.Data[pos : pos+4])
		nals = append(nals, createNAL(n.Data[pos+4:pos+4+size]))

		pos += 4 + size
		if dataSize <= pos {
			return nals
		}
	}
}

func (n *NALUnit) Close() {
	if atomic.CompareAndSwapInt32(&n.closed, 0, 1) {
		runtime.SetFinalizer(n, nil)

		if n.closeFunc != nil {
			n.closeFunc()
		}
	}
}

type NAL struct {
	NALType NALUnitType
	Data    []byte
}

func createNAL(data []byte) NAL {
	return NAL{
		NALType: NALUnitType(((data[0] >> 1) & 0x3f) - 1),
		Data:    data,
	}
}
