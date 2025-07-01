package image

import (
	"github.com/AlexStarov/escpos-GoLang-lib/util"
)

const gs8lMaxY = 831 // Ограничение по высоте для GS 8 L режима

// Raster отправляет изображение в формате растра (bitImage или graphics) на принтер.
func Raster(p Target, width, height, lineWidth int, imgBw []byte, printingType string) {
	if printingType == "bitImage" {
		// GS v 0 режим
		header := []byte{0x1d, 0x76, 0x30, 0}
		imgWidthBytes := (width + 7) >> 3
		header = append(header, util.IntLowHigh(imgWidthBytes, 2)...)
		header = append(header, util.IntLowHigh(height, 2)...)
		payload := append(header, imgBw...)
		p.Raster(width, height, lineWidth, payload, "bitImage")
	} else {
		// GS 8 L — разбиение по высоте
		for l := 0; l < height; {
			lines := gs8lMaxY
			if lines > height-l {
				lines = height - l
			}
			dataBlock := imgBw[l*lineWidth : (l+lines)*lineWidth]

			setup := []byte{
				0x1d, 0x38, 0x4c,
			}
			blockSize := 10 + len(dataBlock)
			setup = append(setup,
				byte(blockSize), byte(blockSize>>8), byte(blockSize>>16), byte(blockSize>>24),
				0x30, 0x70, 0x30,
				0x01, 0x01,
				0x31,
				byte(width), byte(width>>8),
				byte(lines), byte(lines>>8),
			)

			p.Raster(width, lines, lineWidth, append(setup, dataBlock...), "graphics")
			p.Raster(0, 0, 0, []byte{0x1d, 0x28, 0x4c, 0x02, 0x00, 0x30, 0x32}, "graphics")
			l += lines
		}
	}
}
