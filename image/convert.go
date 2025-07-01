package image

import (
	"fmt"
	"image"
	"image/color"
)

type Converter struct {
	MaxWidth  int
	Threshold float64
}

// Print converts an image and sends it to the target
func (c *Converter) Print(img image.Image, target Target) {
	sz := img.Bounds().Size()
	fmt.Printf("Image size: %dx%d\n", sz.X, sz.Y)

	rasterData, realWidth, bytesWidth := c.ToRaster(img)

	mode := "bitImage"
	if sz.Y >= 831 {
		mode = "graphics"
	}

	target.Raster(realWidth, sz.Y, bytesWidth, rasterData, mode)
}

func (c *Converter) ToRaster(img image.Image) ([]byte, int, int) {
	sz := img.Bounds().Size()
	width := sz.X
	if width > c.MaxWidth {
		width = c.MaxWidth
	}

	bytesWidth := (width + 7) / 8
	data := make([]byte, bytesWidth*sz.Y)

	for y := 0; y < sz.Y; y++ {
		for x := 0; x < width; x++ {
			if lightness(img.At(x, y)) <= c.Threshold {
				data[y*bytesWidth+x/8] |= 0x80 >> uint(x%8)
			}
		}
	}

	return data, width, bytesWidth
}

func lightness(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	return float64(55*r+182*g+18*b) / float64(0xffff*(55+182+18))
}
