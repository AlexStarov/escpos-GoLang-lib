//go:build windows
// +build windows

package printer

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// spoolerConn реализует io.ReadWriteCloser поверх Windows Spooler API
type spoolerConn struct {
	hPrinter windows.Handle
}

func (s *spoolerConn) Write(p []byte) (int, error) {
	var written uint32
	r1, _, err := procWritePrinter.Call(
		uintptr(s.hPrinter),
		uintptr(unsafe.Pointer(&p[0])),
		uintptr(len(p)),
		uintptr(unsafe.Pointer(&written)),
	)
	if r1 == 0 {
		return int(written), err
	}
	return int(written), nil
}

func (s *spoolerConn) Read(p []byte) (int, error) {
	// Обычно чтение из принтера через спулер не используется
	return 0, fmt.Errorf("read not supported for Windows spooler connection")
}

func (s *spoolerConn) Close() error {
	procEndPagePrinter.Call(uintptr(s.hPrinter))
	procEndDocPrinter.Call(uintptr(s.hPrinter))
	procClosePrinter.Call(uintptr(unsafe.Pointer(&s.hPrinter)))
	return nil
}

// NewWinPrintSpoolerPrinter создает принтер по имени (Windows Spooler)
func NewWinPrintSpoolerPrinter(printerName string) (*Printer, error) {
	var hPrinter windows.Handle
	pname, _ := windows.UTF16PtrFromString(printerName)
	r1, _, err := procOpenPrinter.Call(
		uintptr(unsafe.Pointer(pname)),
		uintptr(unsafe.Pointer(&hPrinter)),
		0,
	)
	if r1 == 0 {
		return nil, fmt.Errorf("failed to open printer %q: %w", printerName, err)
	}

	// DOC_INFO_1
	docName, _ := windows.UTF16PtrFromString("ESC/POS RAW Document")
	dataType, _ := windows.UTF16PtrFromString("RAW")
	di := docInfo1{
		pDocName:    docName,
		pOutputFile: nil,
		pDatatype:   dataType,
	}

	r1, _, err = procStartDocPrinter.Call(
		uintptr(hPrinter),
		1,
		uintptr(unsafe.Pointer(&di)),
	)
	if r1 == 0 {
		procClosePrinter.Call(uintptr(unsafe.Pointer(&hPrinter)))
		return nil, fmt.Errorf("StartDocPrinter failed: %w", err)
	}

	procStartPagePrinter.Call(uintptr(hPrinter))

	conn := &spoolerConn{hPrinter: hPrinter}
	return NewPrinter(conn)
}

// --- WinAPI binding ---
var (
	modwinspool          = windows.NewLazySystemDLL("winspool.drv")
	procOpenPrinter      = modwinspool.NewProc("OpenPrinterW")
	procClosePrinter     = modwinspool.NewProc("ClosePrinter")
	procStartDocPrinter  = modwinspool.NewProc("StartDocPrinterW")
	procEndDocPrinter    = modwinspool.NewProc("EndDocPrinter")
	procStartPagePrinter = modwinspool.NewProc("StartPagePrinter")
	procEndPagePrinter   = modwinspool.NewProc("EndPagePrinter")
	procWritePrinter     = modwinspool.NewProc("WritePrinter")
)

type docInfo1 struct {
	pDocName    *uint16
	pOutputFile *uint16
	pDatatype   *uint16
}
