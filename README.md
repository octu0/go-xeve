# `go-xeve`

[![License](https://img.shields.io/github/license/octu0/go-xeve)](https://github.com/octu0/go-xeve/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/octu0/go-xeve?status.svg)](https://godoc.org/github.com/octu0/go-xeve)
[![Go Report Card](https://goreportcard.com/badge/github.com/octu0/go-xeve)](https://goreportcard.com/report/github.com/octu0/go-xeve)
[![Releases](https://img.shields.io/github/v/release/octu0/go-xeve)](https://github.com/octu0/go-xeve/releases)

Go bindings for [mpeg5/xeve](https://github.com/mpeg5/xeve)  
MPEG-5 EVC encoder.

## Requirements

requires xeve [install](https://github.com/mpeg5/xeve#how-to-build) on your system

```
$ git clone https://github.com/mpeg5/xeve.git
$ cd xeve
$ mkdir build
$ cd build
$ cmake .. -DSET_PROF=BASE
$ make
$ make install
```

## Usage

### Encode

```go
import "github.com/octu0/go-xeve"

func main() {
	param := createParam(width, height)
	defer param.Close()

	encoder, err := xeve.CreateBaselineEncoder(param)
	if err != nil {
		panic(err)
	}
	defer encoder.Close()

	out := newBitstreamWriter()
	for {
		img, err := loadExampleData()
		if err != nil {
			panic(err)
		}
		nal, err := encoder.Encode(
			img.Y,                    // Y plane
			img.Cb,                   // U plane
			img.Cr,                   // V plane
			img.YStride,              // Y stride
			img.CStride,              // U stride
			img.CStride,              // V stride
			xeve.ColorFormatYCbCr420, // YUV 420
			xeve.BitDepth8,           // 8bit
		)
		if err != nil {
			panic(err)
		}
		defer nal.Close()

		if nal.HasData() != true {
			continue
		}

		out.Write(nal.Data) // write bitstream

		fmt.Printf("Frame:%s Slice:%s Data:%v(%d)\n", nal.NALUnit, nal.Slice, nal.Data[0:10], len(nal.Data))
		// => Frame:IDR Slice:I Data:[0 0 0 21 50 0 128 0 0 0](1234567)

		for idx, nal := range nal.SplitNAL() {
			fmt.Printf("  [%d] type=%s (%d)\n", idx, nal.NALType, len(nal.Data))
		}
		// =>  [0] type=SPS (21)
		// =>  [1] type=PPS (4)
		// =>  [2] type=SEI (1234)
		// =>  [3] type=IDR (1233308)
	}

	nal, err := encoder.Flush()
	if err != nil {
		panic(err)
	}
	defer nal.Close()

	if nal.HasData() {
		out.Write(nal.Data)
		fmt.Printf("[flush] Frame:%s Slice:%s Data:%v(%d)\n", nal.NALUnit, nal.Slice, nal.Data[0:10], len(nal.Data))
		// => [flush] Frame:NonIDR Slice:P Data:[0 0 0 18 3 0 164 172 64 0](22)
	}
}

func createParam(width, height int) *xeve.BaselineParam {
	param, err := xeve.CreateDefaultBaselineParam()
	if err != nil {
		panic(err)
	}

	param.SetPresetTune(xeve.PresetFast, xeve.TuneNone)
	param.SetInputSize(width, height)
	param.SetFramerate(30, 60)
	param.SetBitrate(2000)
	param.SetGOP(xeve.GOPClosed)
	param.SetRateControl(xeve.RateControlABR)
	param.SetBFrames(0)
	return param
}

func loadExampleData() (*image.YCbCr, error) {
	// load YCbCr420 image ...
}

func newBitstreamWriter() io.Writer {
	// bitstream writer ...
}
```
