package printer

import (
	"encoding/base64"
	"fmt"
	"image"
	"io"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/nfnt/resize"

	// Убедитесь, что пакет image импортирован корректно
	imgInternal "github.com/AlexStarov/escpos-GoLang-lib/image" // Убедитесь, что пакет image импортирован корректно
	logInternal "github.com/AlexStarov/escpos-GoLang-lib/log"   // Убедитесь, что пакет image импортирован корректно
	utilInternal "github.com/AlexStarov/escpos-GoLang-lib/util" // Убедитесь, что пакет image импортирован корректно
)

const gs8lMaxY = 831

// Printer wraps sending ESC-POS commands to a io.Writer.
type Printer struct {
	// destination
	w io.ReadWriteCloser // io.ReadWriterCloser allows for both reading and writing, useful for status checks

	// font metrics
	width, height byte

	// state toggles ESC[char]
	underline  byte
	emphasize  byte
	upsidedown byte
	rotate     byte

	// state toggles GS[char]
	reverse, smooth byte

	sync.Mutex
}

// NewPrinter creates a new printer using the specified writer.
func NewPrinter(w io.ReadWriter) (*Printer, error) {
	// Если w реализует ReadWriteCloser, используем его напрямую.
	if rc, ok := w.(io.ReadWriteCloser); ok {
		return &Printer{
			w:      rc,
			width:  1,
			height: 1,
		}, nil
	}
	// Для остальных случаев оборачиваем в nopCloser.
	return &Printer{
		w:      nopCloser{w},
		width:  1,
		height: 1,
	}, nil
}

func (p *Printer) ReadStatus() bool {
	statusOnline := []byte{0x10, 0x04, 0x01}
	p.w.Write(statusOnline)
	time.Sleep(1 * time.Second)
	buf := make([]byte, 1)
	p.w.Read(buf)
	maskOnline := byte(uint(8))
	if len(buf) == 0 {
		return false
	}
	return (buf[0] & maskOnline) == 0
}

// Reset resets the printer state.
func (p *Printer) Reset() {
	p.width = 1
	p.height = 1

	p.underline = 0
	p.emphasize = 0
	p.upsidedown = 0
	p.rotate = 0

	p.reverse = 0
	p.smooth = 0
}
func (p *Printer) CloseConnection() error {
	// return p.w.(net.Conn).Close()
	return p.w.Close()
}

// Write writes buf to printer.
func (p *Printer) Write(buf []byte) (int, error) {
	return p.w.Write(buf)
}

// Init resets the state of the printer, and writes the initialize code.
func (p *Printer) Init() {
	p.Reset()
	p.w.Write([]byte("\x1B@")) // ESC @ (Initialize printer)
}

// End terminates the printer session.
func (p *Printer) End() {
	p.w.Write([]byte("\xFA"))
}

// Cut writes the cut code to the printer.
func (p *Printer) Cut() {
	p.w.Write([]byte("\x1DVA0")) // GS
}

// Cash writes the cash code to the printer.
func (p *Printer) Cash() {
	p.w.Write([]byte("\x1B\x70\x00\x0A\xFF"))
}

// Linefeed writes a line end to the printer.
func (p *Printer) Linefeed() {
	p.w.Write([]byte("\n"))
}

// FormfeedN writes N formfeeds to the printer.
func (p *Printer) FormfeedN(n int) {
	p.w.Write([]byte(fmt.Sprintf("\x1Bd%c", n))) // ESC
}

// Formfeed writes 1 formfeed to the printer.
func (p *Printer) Formfeed() {
	p.FormfeedN(1)
}

// SendFontSize sends the font size command to the printer.
func (p *Printer) SendFontSize() {
	p.w.Write([]byte(fmt.Sprintf("\x1D!%c", ((p.width-1)<<4)|(p.height-1))))
}

// SetFontSize sets the font size state and sends the command to the printer.
func (p *Printer) SetFontSize(width, height byte) {
	if width > 0 && height > 0 && width <= 8 && height <= 8 {
		p.width, p.height = width, height
		p.SendFontSize()
	} else {
		logInternal.Errlog.Printf("Invalid font size passed: %d x %d\n", width, height)
	}
}

// SendUnderline sends the underline command to the printer.
func (p *Printer) SendUnderline() {
	p.w.Write([]byte(fmt.Sprintf("\x1B-%c", p.underline)))
}

// SendEmphasize sends the emphasize / doublestrike command to the printer.
func (p *Printer) SendEmphasize() {
	p.w.Write([]byte(fmt.Sprintf("\x1BG%c", p.emphasize)))
}

// SendUpsidedown sends the upsidedown command to the printer.
func (p *Printer) SendUpsidedown() {
	p.w.Write([]byte(fmt.Sprintf("\x1B{%c", p.upsidedown)))
}

// SendRotate sends the rotate command to the printer.
func (p *Printer) SendRotate() {
	p.w.Write([]byte(fmt.Sprintf("\x1BR%c", p.rotate)))
}

// SendReverse sends the reverse command to the printer.
func (p *Printer) SendReverse() {
	p.w.Write([]byte(fmt.Sprintf("\x1DB%c", p.reverse)))
}

// SendSmooth sends the smooth command to the printer.
func (p *Printer) SendSmooth() {
	p.w.Write([]byte(fmt.Sprintf("\x1Db%c", p.smooth)))
}

// SendMoveX sends the move x command to the printer.
func (p *Printer) SendMoveX(x uint16) {
	p.Write([]byte{0x1b, 0x24, byte(x % 256), byte(x / 256)})
}

// SendMoveY sends the move y command to the printer.
func (p *Printer) SendMoveY(y uint16) {
	p.Write([]byte{0x1d, 0x24, byte(y % 256), byte(y / 256)})
}

// SetUnderline sets the underline state and sends it to the printer.
func (p *Printer) SetUnderline(v byte) {
	p.underline = v
	p.SendUnderline()
}

// SetEmphasize sets the emphasize state and sends it to the printer.
func (p *Printer) SetEmphasize(u byte) {
	p.emphasize = u
	p.SendEmphasize()
}

// SetUpsidedown sets the upsidedown state and sends it to the printer.
func (p *Printer) SetUpsidedown(v byte) {
	p.upsidedown = v
	p.SendUpsidedown()
}

// SetRotate sets the rotate state and sends it to the printer.
func (p *Printer) SetRotate(v byte) {
	p.rotate = v
	p.SendRotate()
}

// SetReverse sets the reverse state and sends it to the printer.
func (p *Printer) SetReverse(v byte) {
	p.reverse = v
	p.SendReverse()
}

// SetSmooth sets the smooth state and sends it to the printer.
func (p *Printer) SetSmooth(v byte) {
	p.smooth = v
	p.SendSmooth()
}

// Pulse sends the pulse (open drawer) code to the printer.
func (p *Printer) Pulse() {
	// with t=2 -- meaning 2*2msec
	p.w.Write([]byte("\x1Bp\x02"))
}

// SetAlign sets the alignment state and sends it to the printer.
func (p *Printer) SetAlign(align string) {
	a := 0
	switch align {
	case "left":
		a = 0
	case "center":
		a = 1
	case "right":
		a = 2
	default:
		log.Printf("Invalid alignment: %s\n", align)
	}
	p.w.Write([]byte(fmt.Sprintf("\x1Ba%c", a)))
}

// Feed feeds the printer, applying the supplied params as necessary.
func (p *Printer) Feed(params map[string]string) error {
	// handle lines (form feed X lines)
	if l, ok := params["line"]; ok {
		if i, err := strconv.Atoi(l); err == nil {
			p.FormfeedN(i)
		} else {
			// log.Fatalf("Invalid line number %s", l)
			return err
		}
	}

	// handle units (dots)
	if u, ok := params["unit"]; ok {
		if i, err := strconv.Atoi(u); err == nil {
			p.SendMoveY(uint16(i))
		} else {
			// log.Fatalf("Invalid unit number %s", u)
			return err
		}
	}

	// send linefeed
	p.Linefeed()

	// reset variables
	p.Reset()

	// reset printer
	p.SendEmphasize()
	p.SendRotate()
	p.SendSmooth()
	p.SendReverse()
	p.SendUnderline()
	p.SendUpsidedown()
	p.SendFontSize()
	p.SendUnderline()

	return nil
}

// FeedAndCut feeds the printer using the supplied params and then sends a cut
// command.
func (p *Printer) FeedAndCut(params map[string]string) {
	if t, ok := params["type"]; ok && t == "feed" {
		p.Formfeed()
	}

	p.Cut()
}

// gSendsend graphics headers.
func (p *Printer) gSend(m byte, fn byte, data []byte) {
	l := len(data) + 2

	p.w.Write([]byte("\x1b(L"))
	p.Write([]byte{byte(l % 256), byte(l / 256), m, fn})
	p.w.Write([]byte(data))
}

// Image writes an image using the supplied params.
func (p *Printer) Image(params map[string]string, data string) error {
	// send alignment to printer
	if align, ok := params["align"]; ok {
		p.SetAlign(align)
	}

	// get width
	wstr, ok := params["width"]
	if !ok {
		log.Println("No width specified on image")
	}

	// get height
	hstr, ok := params["height"]
	if !ok {
		log.Println("No height specified on image")
	}

	// convert width
	width, err := strconv.Atoi(wstr)
	if err != nil {
		// log.Println("Invalid image width %s", wstr)
		return err
	}

	// convert height
	height, err := strconv.Atoi(hstr)
	if err != nil {
		// log.Fatalf("Invalid image height %s", hstr)
		return err
	}

	// decode data frome b64 string
	dec, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		// log.Fatal(err)
		return err
	}

	log.Printf("Image len:%d w: %d h: %d\n", len(dec), width, height)

	header := []byte{
		byte('0'), 0x01, 0x01, byte('1'),
	}

	a := append(header, dec...)

	p.gSend(byte('0'), byte('p'), a)
	p.gSend(byte('0'), byte('2'), []byte{})

	return nil
}

// WriteNode writes a node of type name with the supplied params and data to
// the printer.
func (p *Printer) WriteNode(name string, params map[string]string, data string) {
	cstr := ""
	if data != "" {
		str := data
		if len(data) > 40 {
			str = fmt.Sprintf("%s ...", data[0:40])
		}
		cstr = fmt.Sprintf(" => '%s'", str)
	}
	log.Printf("Write: %s => %+v%s\n", name, params, cstr)

	switch name {
	case "feed":
		p.Feed(params)

	case "cut":
		p.FeedAndCut(params)

	case "pulse":
		p.Pulse()

	case "image":
		p.Image(params, data)
	}
}

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
	if printingType == "bitImage" {
		densityByte := byte(0)
		header := []byte{ // GS v 0 m xL xH yL yH d1...dk
			0x1d, 0x76, 0x30}
		header = append(header, densityByte)
		width = (width + 7) >> 3
		header = append(header, utilInternal.IntLowHigh(width, 2)...)
		header = append(header, utilInternal.IntLowHigh(height, 2)...)

		fullImage := append(header, imgBw...)

		p.Write(fullImage)

	} else if printingType == "graphics" {
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
