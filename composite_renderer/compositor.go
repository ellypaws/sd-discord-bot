package composite_renderer

import (
	"bytes"
	"errors"
	"image"
	"image/draw"
	"image/png"
	"io"
	"math"
)

type compositor struct{}

func (c *compositor) TileImages(imageBufs []io.Reader) (io.Reader, error) {
	numImages := len(imageBufs)
	if numImages == 0 {
		return nil, errors.New("no images provided")
	}

	if numImages == 1 {
		return imageBufs[0], nil
	}

	images := make([]image.Image, numImages)
	var totalWidth, totalHeight int
	for i, buf := range imageBufs {
		img, _, err := image.Decode(buf)
		if err != nil {
			return nil, err
		}
		images[i] = img
		bounds := img.Bounds()
		totalWidth += bounds.Dx()
		totalHeight += bounds.Dy()
	}

	rows, cols := determineLayout(numImages, images)

	canvasWidth, canvasHeight := calculateCanvasSize(images, rows, cols)

	retImage := image.NewRGBA(image.Rect(0, 0, canvasWidth, canvasHeight))

	var x, y, maxHeightInRow int
	for i, img := range images {
		if i%cols == 0 && i != 0 {
			x = 0
			y += maxHeightInRow
			maxHeightInRow = 0
		}

		bounds := img.Bounds()
		maxHeightInRow = max(maxHeightInRow, bounds.Dy())
		draw.Draw(retImage, image.Rect(x, y, x+bounds.Dx(), y+bounds.Dy()), img, bounds.Min, draw.Over)
		x += bounds.Dx()
	}

	imageBuf := new(bytes.Buffer)
	err := png.Encode(imageBuf, retImage)
	if err != nil {
		return nil, err
	}

	return imageBuf, nil
}

func determineLayout(numImages int, images []image.Image) (rows, cols int) {
	if numImages == 1 {
		return 1, 1
	}

	// Basic heuristic: prefer more columns than rows to minimize empty space
	cols = int(math.Ceil(math.Sqrt(float64(numImages))))
	rows = int(math.Ceil(float64(numImages) / float64(cols)))

	// Adjust for aspect ratios
	portraitCount, landscapeCount, squareCount := countImageTypes(images)
	if landscapeCount > portraitCount && landscapeCount > squareCount {
		rows, cols = cols, rows // Prefer wider layout for mostly landscape images
	}

	return
}

func calculateCanvasSize(images []image.Image, rows, cols int) (width, height int) {
	maxWidthPerColumn := make([]int, cols)
	maxHeightPerRow := make([]int, rows)

	for i, img := range images {
		row := i / cols
		col := i % cols
		bounds := img.Bounds()
		maxWidthPerColumn[col] = max(maxWidthPerColumn[col], bounds.Dx())
		maxHeightPerRow[row] = max(maxHeightPerRow[row], bounds.Dy())
	}

	for _, w := range maxWidthPerColumn {
		width += w
	}
	for _, h := range maxHeightPerRow {
		height += h
	}

	return
}

func countImageTypes(images []image.Image) (portrait, landscape, square int) {
	for _, img := range images {
		bounds := img.Bounds()
		width := bounds.Dx()
		height := bounds.Dy()

		if width > height {
			landscape++
		} else if width < height {
			portrait++
		} else {
			square++
		}
	}
	return
}
