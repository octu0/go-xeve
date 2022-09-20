package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"

	"github.com/octu0/go-xeve"
)

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

func main() {
	var width, height int
	var output bool
	flag.IntVar(&width, "width", 320, "input image width")
	flag.IntVar(&height, "height", 240, "input image height")
	flag.BoolVar(&output, "output", false, "output bitstream")
	flag.Parse()

	param := createParam(width, height)
	defer param.Close()

	encoder, err := xeve.CreateBaselineEncoder(param)
	if err != nil {
		panic(err)
	}
	defer encoder.Close()

	var out *os.File
	if output {
		out, err = os.Create("/tmp/out.evc")
		if err != nil {
			panic(err)
		}
		defer out.Close()

		fmt.Printf("bitstream write to %s\n", out.Name())
	}

	num := 0
	for i := 0; i < 120; i += 1 {
		img, err := loadExampleData(i)
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

		fmt.Printf("[%d] Frame:%s Slice:%s Data:%v(%d)\n", num, nal.NALUnit, nal.Slice, nal.Data[0:10], len(nal.Data))

		for idx, nal := range nal.SplitNAL() {
			fmt.Printf("  [%d] type=%s (%d)\n", idx, nal.NALType, len(nal.Data))
		}

		if output {
			_, err := out.Write(nal.Data)
			if err != nil {
				panic(err)
			}
		}
		num += 1
	}

	nal, err := encoder.Flush()
	if err != nil {
		panic(err)
	}
	defer nal.Close()

	if nal.HasData() {
		fmt.Printf("[flush] Frame:%s Slice:%s Data:%v(%d)\n", nal.NALUnit, nal.Slice, nal.Data[0:10], len(nal.Data))
		if output {
			out.Write(nal.Data)
		}
	}

	if output {
		out.Sync()
	}
}

func loadExampleData(i int) (*image.YCbCr, error) {
	path := fmt.Sprintf("./testdata/src_%02d.png", i%16)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	rgba, err := pngToRGBA(data)
	if err != nil {
		return nil, err
	}

	img := image.NewYCbCr(rgba.Bounds(), image.YCbCrSubsampleRatio420)
	if err := rgbaToYCbCrImage(img, rgba); err != nil {
		return nil, err
	}
	return img, nil
}

func pngToRGBA(data []byte) (*image.RGBA, error) {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if i, ok := img.(*image.RGBA); ok {
		return i, nil
	}

	b := img.Bounds()
	rgba := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y += 1 {
		for x := b.Min.X; x < b.Max.X; x += 1 {
			c := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
			rgba.Set(x, y, c)
		}
	}
	return rgba, nil
}

func rgbaToYCbCrImage(dst *image.YCbCr, src *image.RGBA) error {
	rect := src.Bounds()
	width, height := rect.Dx(), rect.Dy()

	for w := 0; w < width; w += 1 {
		for h := 0; h < height; h += 1 {
			c := src.RGBAAt(w, h)
			y, u, v := color.RGBToYCbCr(c.R, c.G, c.B)
			dst.Y[dst.YOffset(w, h)] = y
			dst.Cb[dst.COffset(w, h)] = u
			dst.Cr[dst.COffset(w, h)] = v
		}
	}
	return nil
}
