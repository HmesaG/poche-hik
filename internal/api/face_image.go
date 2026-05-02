package api

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"net/http"
)

const (
	targetFaceWidth  = 480
	targetFaceHeight = 640
	minFaceSide      = 160
	maxFaceBytes     = 512 << 10
)

type preparedFaceMeta struct {
	Width  int
	Height int
	Bytes  int
}

func prepareFaceImage(imageData []byte) ([]byte, preparedFaceMeta, error) {
	contentType := detectImageContentType(imageData)

	var (
		src image.Image
		err error
	)

	switch contentType {
	case "image/jpeg":
		src, err = jpeg.Decode(bytes.NewReader(imageData))
	case "image/png":
		src, err = png.Decode(bytes.NewReader(imageData))
	default:
		return nil, preparedFaceMeta{}, fmt.Errorf("only JPEG or PNG images are allowed")
	}
	if err != nil {
		return nil, preparedFaceMeta{}, fmt.Errorf("invalid image file")
	}

	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width < minFaceSide || height < minFaceSide {
		return nil, preparedFaceMeta{}, fmt.Errorf("image too small; use at least %dx%d pixels", minFaceSide, minFaceSide)
	}

	cropped := cropToAspect(src, targetFaceWidth, targetFaceHeight)
	scaled := resizeNearest(cropped, targetFaceWidth, targetFaceHeight)

	qualities := []int{92, 88, 82, 76}
	for _, quality := range qualities {
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, scaled, &jpeg.Options{Quality: quality}); err != nil {
			return nil, preparedFaceMeta{}, fmt.Errorf("failed to encode JPEG")
		}
		if buf.Len() <= maxFaceBytes || quality == qualities[len(qualities)-1] {
			return buf.Bytes(), preparedFaceMeta{
				Width:  targetFaceWidth,
				Height: targetFaceHeight,
				Bytes:  buf.Len(),
			}, nil
		}
	}

	return nil, preparedFaceMeta{}, fmt.Errorf("failed to prepare image")
}

func detectImageContentType(imageData []byte) string {
	if len(imageData) == 0 {
		return ""
	}
	return http.DetectContentType(imageData)
}

func cropToAspect(src image.Image, targetW, targetH int) image.Image {
	b := src.Bounds()
	srcW := b.Dx()
	srcH := b.Dy()
	targetRatio := float64(targetW) / float64(targetH)
	srcRatio := float64(srcW) / float64(srcH)

	crop := b
	if srcRatio > targetRatio {
		newW := int(math.Round(float64(srcH) * targetRatio))
		offsetX := (srcW - newW) / 2
		crop = image.Rect(b.Min.X+offsetX, b.Min.Y, b.Min.X+offsetX+newW, b.Max.Y)
	} else if srcRatio < targetRatio {
		newH := int(math.Round(float64(srcW) / targetRatio))
		offsetY := (srcH - newH) / 2
		crop = image.Rect(b.Min.X, b.Min.Y+offsetY, b.Max.X, b.Min.Y+offsetY+newH)
	}

	dst := image.NewRGBA(image.Rect(0, 0, crop.Dx(), crop.Dy()))
	for y := 0; y < crop.Dy(); y++ {
		for x := 0; x < crop.Dx(); x++ {
			dst.Set(x, y, src.At(crop.Min.X+x, crop.Min.Y+y))
		}
	}
	return dst
}

func resizeNearest(src image.Image, width, height int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	b := src.Bounds()
	srcW := b.Dx()
	srcH := b.Dy()
	if srcW == 0 || srcH == 0 {
		fillRGBA(dst, color.RGBA{0, 0, 0, 255})
		return dst
	}

	for y := 0; y < height; y++ {
		srcY := b.Min.Y + (y * srcH / height)
		for x := 0; x < width; x++ {
			srcX := b.Min.X + (x * srcW / width)
			dst.Set(x, y, src.At(srcX, srcY))
		}
	}
	return dst
}

func fillRGBA(img *image.RGBA, c color.RGBA) {
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			img.SetRGBA(x, y, c)
		}
	}
}
