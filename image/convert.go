package image

import (
	"fmt"
	"image"
	"image/color"

	logInternal "github.com/AlexStarov/escpos-GoLang-lib/log" // Убедитесь, что пакет image импортирован корректно
)

const gs8lMaxY = 831

type Converter struct {
	// The maximum line width of the printer, in dots
	MaxWidth int

	// The threashold between white and black dots
	Threshold float64
}

func (c *Converter) Print(img image.Image, target Target) {
	sz := img.Bounds().Size()
	logInternal.LogMessage(logInternal.INFO, fmt.Sprintf("sz: %+v, X: %d, Y: %d", sz, sz.X, sz.Y))

	data, rw, bw := c.ToRaster(img)

	mode := "bitImage"
	if sz.Y >= gs8lMaxY {
		mode = "graphics"
	}

	target.Raster(rw, sz.Y, bw, data, mode)
}
func (c *Converter) ToRaster(img image.Image) (data []byte, imageWidth, bytesWidth int) {
	sz := img.Bounds().Size()

	// lines are packed in bits
	imageWidth = sz.X
	if imageWidth > c.MaxWidth {
		// truncate if image is too large
		imageWidth = c.MaxWidth
	}

	bytesWidth = imageWidth / 8
	if imageWidth%8 != 0 {
		bytesWidth += 1
	}

	data = make([]byte, bytesWidth*sz.Y)

	for y := 0; y < sz.Y; y++ {
		for x := 0; x < imageWidth; x++ {
			if lightness(img.At(x, y)) <= c.Threshold {
				// position in data is: line_start + x / 8
				// line_start is y * bytesWidth
				// then 8 bits per byte
				data[y*bytesWidth+x/8] |= 0x80 >> uint(x%8)
			}
		}
	}

	return
}

const (
	lumR, lumG, lumB = 55, 182, 18
	// gs8lMaxY         = 1662
	// gs8lMaxY = 831
)

func lightness(c color.Color) float64 {
	r, g, b, _ := c.RGBA()

	return float64(lumR*r+lumG*g+lumB*b) / float64(0xffff*(lumR+lumG+lumB))
}
