package printer

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	logInternal "github.com/AlexStarov/escpos-GoLang-lib/log"
)

const gs8lMaxY = 831

type Printer struct {
	t Transport

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

func NewPrinter(w io.ReadWriter) (*Printer, error) {
    var transport Transport

    // Если нам дали сетевое соединение — проверим порт
    if conn, ok := w.(net.Conn); ok {
        addr := conn.RemoteAddr().String()
        // LPD соединение (порт 515): используем буферизованный LPDTransport
        if strings.HasSuffix(addr, ":515") {
            transport = NewLPDTransport(conn, "lp")
        } else {
            // Любой другой порт — просто raw passthrough
            transport = &RawTransport{conn: conn}
        }
    } else if rc, ok := w.(io.ReadWriteCloser); ok {
        // Не сетевой conn — трактуем как RAW
        transport = &RawTransport{conn: rc}
    } else {
        // Любой io.ReadWriter (например, bytes.Buffer) — оборачиваем в nopCloser и RAW
        transport = &RawTransport{conn: nopCloser{w}}
    }

    return &Printer{
        t:      transport,
        width:  1,
        height: 1,
    }, nil
}

func (p *Printer) ReadStatus() bool {
	statusOnline := []byte{0x10, 0x04, 0x01}
	p.t.Write(statusOnline)
	time.Sleep(1 * time.Second)
	buf := make([]byte, 1)
	p.t.Read(buf)
	maskOnline := byte(uint(8))
	if len(buf) == 0 {
		return false
	}
	return (buf[0] & maskOnline) == 0
}

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
	return p.t.Close()
}

func (p *Printer) Write(buf []byte) (int, error) {
	return p.t.Write(buf)
}

func (p *Printer) Init() {
	p.Reset()
	p.t.Write([]byte("\x1B@")) // ESC @ (Initialize printer)
}

func (p *Printer) End() {
	p.t.Write([]byte("\xFA"))
}

func (p *Printer) Cut() {
	p.t.Write([]byte("\x1DVA0")) // GS
}

func (p *Printer) Cash() {
	p.t.Write([]byte("\x1B\x70\x00\x0A\xFF"))
}

func (p *Printer) Linefeed() {
	p.t.Write([]byte("\n"))
}

func (p *Printer) FormfeedN(n int) {
	p.t.Write([]byte(fmt.Sprintf("\x1Bd%c", n))) // ESC
}

func (p *Printer) Formfeed() {
	p.FormfeedN(1)
}

func (p *Printer) SendFontSize() {
	p.t.Write([]byte(fmt.Sprintf("\x1D!%c", ((p.width-1)<<4)|(p.height-1))))
}

func (p *Printer) SetFontSize(width, height byte) {
	if width > 0 && height > 0 && width <= 8 && height <= 8 {
		p.width, p.height = width, height
		p.SendFontSize()
	} else {
		logInternal.Errlog.Printf("Invalid font size passed: %d x %d\n", width, height)
	}
}

func (p *Printer) SendUnderline() {
	p.t.Write([]byte(fmt.Sprintf("\x1B-%c", p.underline)))
}

func (p *Printer) SendEmphasize() {
	p.t.Write([]byte(fmt.Sprintf("\x1BG%c", p.emphasize)))
}

func (p *Printer) SendUpsidedown() {
	p.t.Write([]byte(fmt.Sprintf("\x1B{%c", p.upsidedown)))
}

func (p *Printer) SendRotate() {
	p.t.Write([]byte(fmt.Sprintf("\x1BR%c", p.rotate)))
}

func (p *Printer) SendReverse() {
	p.t.Write([]byte(fmt.Sprintf("\x1DB%c", p.reverse)))
}

func (p *Printer) SendSmooth() {
	p.t.Write([]byte(fmt.Sprintf("\x1Db%c", p.smooth)))
}

func (p *Printer) SendMoveX(x uint16) {
	p.Write([]byte{0x1b, 0x24, byte(x % 256), byte(x / 256)})
}

func (p *Printer) SendMoveY(y uint16) {
	p.Write([]byte{0x1d, 0x24, byte(y % 256), byte(y / 256)})
}

func (p *Printer) SetUnderline(v byte) {
	p.underline = v
	p.SendUnderline()
}

func (p *Printer) SetEmphasize(u byte) {
	p.emphasize = u
	p.SendEmphasize()
}

func (p *Printer) SetUpsidedown(v byte) {
	p.upsidedown = v
	p.SendUpsidedown()
}

func (p *Printer) SetRotate(v byte) {
	p.rotate = v
	p.SendRotate()
}

func (p *Printer) SetReverse(v byte) {
	p.reverse = v
	p.SendReverse()
}

func (p *Printer) SetSmooth(v byte) {
	p.smooth = v
	p.SendSmooth()
}

func (p *Printer) Pulse() {
	// with t=2 -- meaning 2*2msec
	p.t.Write([]byte("\x1Bp\x02"))
}

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
	p.t.Write([]byte(fmt.Sprintf("\x1Ba%c", a)))
}

func (p *Printer) Feed(params map[string]string) error {
	// handle lines (form feed X lines)
	if l, ok := params["line"]; ok {
		if i, err := strconv.Atoi(l); err == nil {
			p.FormfeedN(i)
		} else {
			return err
		}
	}

	// handle units (dots)
	if u, ok := params["unit"]; ok {
		if i, err := strconv.Atoi(u); err == nil {
			p.SendMoveY(uint16(i))
		} else {
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

func (p *Printer) FeedAndCut(params map[string]string) {
	if t, ok := params["type"]; ok && t == "feed" {
		p.Formfeed()
	}
	p.Cut()
}

func (p *Printer) gSend(m byte, fn byte, data []byte) {
	l := len(data) + 2

	p.t.Write([]byte("\x1b(L"))
	p.Write([]byte{byte(l % 256), byte(l / 256), m, fn})
	p.t.Write([]byte(data))
}

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
		return err
	}

	// convert height
	height, err := strconv.Atoi(hstr)
	if err != nil {
		return err
	}

	// decode data from b64 string
	dec, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
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
