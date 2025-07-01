package printer

import (
	"fmt"
	"image"
	"io"
	"os"
	"time"

	"github.com/nfnt/resize"

	img "github.com/AlexStarov/escpos-GoLang-lib/image" // Убедитесь, что пакет image импортирован корректно
)

type Printer struct {
	w          io.ReadWriteCloser
	width      byte
	height     byte
	underline  byte
	emphasize  byte
	upsidedown byte
	rotate     byte
	reverse    byte
	smooth     byte
}

func NewPrinter(w io.ReadWriter) (*Printer, error) {
	if closer, ok := w.(io.ReadWriteCloser); ok {
		return &Printer{w: closer, width: 1, height: 1}, nil
	}
	return &Printer{w: nopCloser{w}, width: 1, height: 1}, nil
}

func (p *Printer) Init() {
	p.Reset()
	p.w.Write([]byte("\x1B@"))
}

func (p *Printer) Reset() {
	p.width, p.height = 1, 1
	p.underline, p.emphasize, p.upsidedown, p.rotate = 0, 0, 0, 0
	p.reverse, p.smooth = 0, 0
}

func (p *Printer) Write(data []byte) (int, error) {
	return p.w.Write(data)
}

func (p *Printer) Cut() {
	p.w.Write([]byte("\x1DVA0"))
}

func (p *Printer) Cash() {
	p.w.Write([]byte("\x1B\x70\x00\x0A\xFF"))
}

func (p *Printer) Pulse() {
	p.w.Write([]byte("\x1Bp\x02"))
}

func (p *Printer) Linefeed() {
	p.w.Write([]byte("\n"))
}

func (p *Printer) SendFontSize() {
	p.w.Write([]byte(fmt.Sprintf("\x1D!%c", ((p.width-1)<<4)|(p.height-1))))
}

func (p *Printer) SetFontSize(width, height byte) {
	if width > 0 && height > 0 {
		p.width, p.height = width, height
		p.SendFontSize()
	}
}

func (p *Printer) SetAlign(align string) {
	code := byte(0)
	if align == "center" {
		code = 1
	} else if align == "right" {
		code = 2
	}
	p.w.Write([]byte(fmt.Sprintf("\x1Ba%c", code)))
}

// PrintImage печатает изображение из файла
func (p *Printer) PrintImage(imgPath string) error {
	file, err := os.Open(imgPath)
	if err != nil {
		return err
	}
	defer file.Close()

	imgDecoded, _, err := image.Decode(file)
	if err != nil {
		return err
	}

	config, _, _ := image.DecodeConfig(file)
	if config.Height > 450 || config.Width > 450 {
		imgDecoded = resize.Resize(450, 0, imgDecoded, resize.Lanczos3)
	}

	converter := &img.Converter{
		MaxWidth:  512,
		Threshold: 0.5,
	}
	p.SetAlign("center")
	converter.Print(imgDecoded, p)

	time.Sleep(100 * time.Millisecond)
	return nil
}

func (p *Printer) Raster(width, height, bytesWidth int, rasterData []byte, printingType string) {
	p.Write(rasterData)
}

type nopCloser struct {
	io.ReadWriter
}

func (nopCloser) Close() error {
	return nil
}

func (p *Printer) CloseConnection() error {
	if p.w != nil {
		if closer, ok := p.w.(io.Closer); ok {
			return closer.Close()
		}
	}
	return nil
}

// func (p *Printer) CloseConnection() error {
// 	// return p.w.(net.Conn).Close()
// 	return p.w.Close()
// }
