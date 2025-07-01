package printer

import (
    "fmt"
    "io"
)

type Printer struct {
    w           io.ReadWriteCloser
    width       byte
    height      byte
    underline   byte
    emphasize   byte
    upsidedown  byte
    rotate      byte
    reverse     byte
    smooth      byte
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

type nopCloser struct {
    io.ReadWriter
}

func (nopCloser) Close() error {
    return nil
}
