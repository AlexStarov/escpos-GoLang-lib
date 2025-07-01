package printer

import (
	"fmt"

	"github.com/google/gousb"
)

type usbConn struct {
	ctx  *gousb.Context
	dev  *gousb.Device
	cfg  *gousb.Config
	intf *gousb.Interface
	out  *gousb.OutEndpoint
	in   *gousb.InEndpoint
}

func NewUSBPrinter(vendorID, productID gousb.ID) (*Printer, error) {
	ctx := gousb.NewContext()
	dev, err := findUSBPrinter(ctx, vendorID, productID)
	if err != nil {
		ctx.Close()
		return nil, err
	}

	dev.SetAutoDetach(true)
	cfg, err := dev.Config(1)
	if err != nil {
		dev.Close()
		ctx.Close()
		return nil, err
	}

	intf, err := cfg.Interface(0, 0)
	if err != nil {
		cfg.Close()
		dev.Close()
		ctx.Close()
		return nil, err
	}

	outEp, err := intf.OutEndpoint(0x01)
	if err != nil {
		intf.Close()
		cfg.Close()
		dev.Close()
		ctx.Close()
		return nil, err
	}

	inEp, err := intf.InEndpoint(1)
	if err != nil {
		inEp = nil
	}

	conn := &usbConn{ctx, dev, cfg, intf, outEp, inEp}
	printer, err := NewPrinter(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return printer, nil
}

func (u *usbConn) Read(p []byte) (int, error) {
	if u.in != nil {
		return u.in.Read(p)
	}
	return 0, fmt.Errorf("USB read not supported")
}

func (u *usbConn) Write(p []byte) (int, error) {
	return u.out.Write(p)
}

func (u *usbConn) Close() error {
	if u.intf != nil {
		u.intf.Close()
	}
	if u.cfg != nil {
		u.cfg.Close()
	}
	if u.dev != nil {
		u.dev.Close()
	}
	if u.ctx != nil {
		u.ctx.Close()
	}
	return nil
}
