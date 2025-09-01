package printer

import (
	"fmt"
	"image"
	"log"
	"os"

	"github.com/nfnt/resize"

	// Убедитесь, что пакет image импортирован корректно
	imgInternal "github.com/AlexStarov/escpos-GoLang-lib/image" // Убедитесь, что пакет image импортирован корректно
	logInternal "github.com/AlexStarov/escpos-GoLang-lib/log"   // Убедитесь, что пакет image импортирован корректно
	utilInternal "github.com/AlexStarov/escpos-GoLang-lib/util" // Убедитесь, что пакет image импортирован корректно
)

// PrintImage Print Image
func (p *Printer) PrintImage(imgPath string) error {
	imgFile, err := os.Open(imgPath)
	if err != nil {
		// log.Fatal(err)
		return err
	}

	img, imgFormat, err := image.Decode(imgFile)
	imgConfig, _, _ := image.DecodeConfig(imgFile)
	if imgConfig.Height > 450 || imgConfig.Width > 450 {
		img = resize.Resize(450, 0, img, resize.Lanczos3)
	}
	imgFile.Close()
	if err != nil {
		// log.Fatal(err)
		return err
	}
	log.Print("Loaded image, format: ", imgFormat)

	rasterConv := &imgInternal.Converter{
		MaxWidth:  512,
		Threshold: 0.5,
	}
	p.SetAlign("center")
	rasterConv.Print(img, p)
	return nil
}

// Raster writes a rasterized version of a black and white image to the printer
// with the specified width, height, and lineWidth bytes per line.
func (p *Printer) Raster(width, height, lineWidth int, imgBw []byte, printingType string) {

	// std1log.Printf("width: %d, height: %d, lineWidth: %d\n", width, height, lineWidth)
	switch printingType {
	case "bitImage":
		densityByte := byte(0)
		header := []byte{ // GS v 0 m xL xH yL yH d1...dk
			0x1d, 0x76, 0x30}
		header = append(header, densityByte)
		width = (width + 7) >> 3
		header = append(header, utilInternal.IntLowHigh(width, 2)...)
		header = append(header, utilInternal.IntLowHigh(height, 2)...)

		fullImage := append(header, imgBw...)

		p.Write(fullImage)

	case "graphics":
		for l := 0; l < height; {

			logInternal.LogMessage(logInternal.INFO, fmt.Sprintf("graphics --->>> l: %d, height: %d", l, height))

			lines := gs8lMaxY
			if lines > height-l {
				lines = height - l
			}

			f112P := 10 + lines*lineWidth

			p.Write([]byte{
				0x1d, 0x38, 0x4c, // GS 8 L, Store the graphics data in the print buffer -- (raster format), p. 252
				byte(f112P), byte(f112P >> 8), byte(f112P >> 16), byte(f112P >> 24), // p1 p2 p3 p4
				0x30, 0x70, 0x30, // function 112
				0x01, 0x01, // bx, by -- zoom
				0x31,                          // c -- single-color printing model
				byte(width), byte(width >> 8), // xl, xh -- number of dots in the horizontal direction
				byte(lines), byte(lines >> 8), // yl, yh -- number of dots in the vertical direction
			})

			// write line
			p.Write(imgBw[l*lineWidth : (l+lines)*lineWidth])

			p.Write([]byte{
				0x1d, 0x28, 0x4c, 0x02, 0x00, 0x30,
				0x32, //  Fn 50
			})

			l += lines
		}
	}
}
