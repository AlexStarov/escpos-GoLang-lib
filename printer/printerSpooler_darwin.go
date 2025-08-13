//go:build darwin
// +build darwin

package printer

import "fmt"

// NewWinPrintSpoolerPrinter — заглушка для macOS
func NewWinPrintSpoolerPrinter(printerName string) (*Printer, error) {
    return nil, fmt.Errorf("Windows Spooler printing is only supported on Windows")
}
