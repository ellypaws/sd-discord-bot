package composite_renderer

import (
	"bytes"
	"errors"
	"image"
	"image/draw"
	"image/png"
	"math"
)

type tilerImpl struct{}

func (r *tilerImpl) TileImages(imageBufs []*bytes.Buffer) (*bytes.Buffer, error) {
	numImages := len(imageBufs)
	if numImages == 0 {
		return nil, errors.New("no images provided")
	}

	images := make([]image.Image, numImages)
	for i, buf := range imageBufs {
		img, _, err := image.Decode(buf)
		if err != nil {
			return nil, err
		}
		images[i] = img
	}

	firstBounds := images[0].Bounds()
	for _, img := range images {
		if img.Bounds() != firstBounds {
			return nil, errors.New("images are not the same size")
		}
	}

	sideLen := int(math.Ceil(math.Sqrt(float64(numImages))))
	canvasWidth := firstBounds.Max.X * sideLen
	canvasHeight := firstBounds.Max.Y * sideLen
	retImage := image.NewRGBA(image.Rect(0, 0, canvasWidth, canvasHeight))

	for i, img := range images {
		x := (i % sideLen) * firstBounds.Max.X
		y := (i / sideLen) * firstBounds.Max.Y
		draw.Draw(retImage, img.Bounds().Add(image.Pt(x, y)), img, image.Point{}, draw.Over)
	}

	imageBuf := new(bytes.Buffer)
	err := png.Encode(imageBuf, retImage)
	if err != nil {
		return nil, err
	}

	return imageBuf, nil
}
