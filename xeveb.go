package xeve

/*
#cgo CFLAGS: -I${SRCDIR}/include -I/usr/local/include -I/usr/include -I/usr/local/include/xeveb
#cgo LDFLAGS: -L${SRCDIR} -L/usr/local/lib -L/usr/lib -lxeveb -lm -ldl
#include <stdint.h>
#include <stdlib.h>

#include "xeveb.h"
*/
import "C"

import (
	"bytes"
	"fmt"
	"runtime"
	"sync/atomic"
	"unsafe"
)

const (
	defaultMaxBitstreamBufferSize int = 10 * 1024 * 1024
)

type BaselineParam struct {
	paramPtr               unsafe.Pointer // *XEVE_PARAM
	maxBitstreamBufferSize int
	closed                 int32
}

func (p *BaselineParam) SetMaxBitstreamBufferSize(size int) bool {
	p.maxBitstreamBufferSize = size
	return true
}

func (p *BaselineParam) SetPresetTune(preset PresetType, tune TuneType) bool {
	ret := int(C.xeveb_param_set_preset_tune(
		(*C.XEVE_PARAM)(p.paramPtr),
		C.uchar(preset),
		C.uchar(tune),
	))
	rc := ReturnCode(ret)
	return Succeed(rc)
}

func (p *BaselineParam) SetInputSize(width, height int) bool {
	ret := int(C.xeveb_param_set_input_size(
		(*C.XEVE_PARAM)(p.paramPtr),
		C.int(width),
		C.int(height),
	))
	rc := ReturnCode(ret)
	return Succeed(rc)
}

func (p *BaselineParam) SetFramerate(fps, keyint int) bool {
	ret := int(C.xeveb_param_set_framerate(
		(*C.XEVE_PARAM)(p.paramPtr),
		C.int(fps),
		C.int(keyint),
	))
	rc := ReturnCode(ret)
	return Succeed(rc)
}

func (p *BaselineParam) SetRateControl(rc RateControlType) bool {
	ret := int(C.xeveb_param_set_ratecontrol(
		(*C.XEVE_PARAM)(p.paramPtr),
		C.int(rc),
	))
	return Succeed(ReturnCode(ret))
}

// bitrate (unit: kbps)
func (p *BaselineParam) SetBitrate(bitrate int) bool {
	ret := int(C.xeveb_param_set_bitrate(
		(*C.XEVE_PARAM)(p.paramPtr),
		C.int(bitrate),
	))
	rc := ReturnCode(ret)
	return Succeed(rc)
}

func (p *BaselineParam) SetGOP(gop GOPType) bool {
	ret := int(C.xeveb_param_set_gop(
		(*C.XEVE_PARAM)(p.paramPtr),
		C.int(gop),
	))
	rc := ReturnCode(ret)
	return Succeed(rc)
}

func (p *BaselineParam) SetBFrames(size int) bool {
	ret := int(C.xeveb_param_set_bframes(
		(*C.XEVE_PARAM)(p.paramPtr),
		C.int(size),
	))
	rc := ReturnCode(ret)
	return Succeed(rc)
}

func (p *BaselineParam) SetUseAnnexB(enable bool) bool {
	use := 0
	if enable {
		use = 1
	}
	ret := int(C.xeveb_param_set_use_annexb(
		(*C.XEVE_PARAM)(p.paramPtr),
		C.int(use),
	))
	rc := ReturnCode(ret)
	return Succeed(rc)
}

func (p *BaselineParam) DebugDump() string {
	return fmt.Sprintf("%+v", (*C.XEVE_PARAM)(p.paramPtr))
}

func (p *BaselineParam) Close() {
	if atomic.CompareAndSwapInt32(&p.closed, 0, 1) {
		runtime.SetFinalizer(p, nil)

		C.xeveb_free_xeve_param(
			(*C.XEVE_PARAM)(p.paramPtr),
		)
	}
}

func finalizeBaselineParam(p *BaselineParam) {
	p.Close()
}

func CreateDefaultBaselineParam() (*BaselineParam, error) {
	ret := unsafe.Pointer(C.xeveb_default_param())
	if ret == nil {
		return nil, fmt.Errorf("failed to call xeveb_default_param()")
	}

	p := &BaselineParam{
		paramPtr:               ret,
		maxBitstreamBufferSize: defaultMaxBitstreamBufferSize,
		closed:                 0,
	}
	runtime.SetFinalizer(p, finalizeBaselineParam)
	return p, nil
}

type BaselineEncoder struct {
	id     unsafe.Pointer // XEVE
	param  *BaselineParam
	bitb   unsafe.Pointer // *XEVE_BITB
	closed int32
	bumped int32
}

func (e *BaselineEncoder) isClosed() bool {
	return atomic.LoadInt32(&e.closed) == 1
}

func (e *BaselineEncoder) isBumped() bool {
	return atomic.LoadInt32(&e.bumped) == 1
}

func (e *BaselineEncoder) encode() (*NALUnit, error) {
	ret := unsafe.Pointer(C.xeveb_encode(
		(C.XEVE)(e.id),
		(*C.XEVE_BITB)(e.bitb),
	))
	if ret == nil {
		return nil, fmt.Errorf("failed to call xeveb_encode()")
	}

	result := (*C.xeveb_encode_result_t)(ret)
	defer C.xeveb_free_result(result)

	if Ok != int(result.status) {
		return &NALUnit{}, nil
	}

	return e.copyNALUnit(result)
}

func (e *BaselineEncoder) Encode(y, u, v []byte, strideY, strideU, strideV int, colorFormat ColorFormatType, bitDepth BitDepthType) (*NALUnit, error) {
	if e.isClosed() {
		return nil, fmt.Errorf("encoder closed")
	}

	imgb := unsafe.Pointer(C.xeveb_create_imgb(
		(*C.XEVE_PARAM)(e.param.paramPtr),
		(*C.uchar)(unsafe.Pointer(&y[0])),
		(*C.uchar)(unsafe.Pointer(&u[0])),
		(*C.uchar)(unsafe.Pointer(&v[0])),
		C.int(strideY),
		C.int(strideU),
		C.int(strideV),
		C.int(len(y)),
		C.int(len(u)),
		C.int(len(v)),
		C.uchar(colorFormat),
		C.uchar(bitDepth),
	))
	if imgb == nil {
		return nil, fmt.Errorf("failed to call xeveb_create_imgb()")
	}
	defer C.xeveb_free_imgb((*C.XEVE_IMGB)(imgb))

	ret := int(C.xeveb_push(
		(C.XEVE)(e.id),
		(*C.XEVE_IMGB)(imgb),
	))
	if Failed(ReturnCode(ret)) {
		return &NALUnit{}, fmt.Errorf("failed to call xeveb_push()")
	}
	return e.encode()
}

func (e *BaselineEncoder) Flush() (*NALUnit, error) {
	if e.isClosed() {
		return nil, fmt.Errorf("encoder closed")
	}

	if atomic.CompareAndSwapInt32(&e.bumped, 0, 1) != true {
		return nil, fmt.Errorf("already bumped")
	}

	ret := int(C.xeveb_bump(
		(C.XEVE)(e.id),
		(*C.XEVE_BITB)(e.bitb),
	))
	if Failed(ReturnCode(ret)) {
		return &NALUnit{}, fmt.Errorf("failed to call xeveb_bump()")
	}
	return e.encode()
}

func (e *BaselineEncoder) copyNALUnit(result *C.xeveb_encode_result_t) (*NALUnit, error) {
	buf := pool.Get().(*bytes.Buffer)
	buf.Reset()
	buf.Write(C.GoBytes(unsafe.Pointer(result.data), result.size))

	nal := &NALUnit{
		NALUnit:   NALUnitType(uint8(result.nalu_type)),
		Slice:     SliceType(int8(result.slice_type)),
		Data:      buf.Bytes(),
		closeFunc: createReleasePoolFunc(buf),
		closed:    0,
	}
	nal.setFinalizer()
	return nal, nil
}

func (e *BaselineEncoder) Close() {
	if atomic.CompareAndSwapInt32(&e.closed, 0, 1) {
		runtime.SetFinalizer(e, nil)

		// bump not yet
		if atomic.CompareAndSwapInt32(&e.bumped, 0, 1) {
			C.xeveb_free_xeve(
				(C.XEVE)(e.id),
			)
		}

		C.xeveb_free_bitb(
			(*C.XEVE_BITB)(e.bitb),
		)
		e.param.Close()
	}
}

func finalizeBaselineEncoder(e *BaselineEncoder) {
	e.Close()
}

func CreateBaselineEncoder(param *BaselineParam) (*BaselineEncoder, error) {
	bitb := unsafe.Pointer(C.xeveb_create_bitb(
		C.int(param.maxBitstreamBufferSize),
	))
	if bitb == nil {
		return nil, fmt.Errorf("failed to call xeveb_create_bitb()")
	}

	id := unsafe.Pointer(C.xeveb_create(
		(*C.XEVE_PARAM)(param.paramPtr),
		C.int(param.maxBitstreamBufferSize),
	))
	if id == nil {
		C.xeveb_free_bitb(
			(*C.XEVE_BITB)(bitb),
		)
		return nil, fmt.Errorf("failed to call xeveb_create()")
	}

	e := &BaselineEncoder{
		id:     id,
		param:  param,
		bitb:   bitb,
		closed: 0,
		bumped: 0,
	}
	runtime.SetFinalizer(e, finalizeBaselineEncoder)
	return e, nil
}
