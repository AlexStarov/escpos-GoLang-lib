package printer

import (
    "fmt"
    "time"

    "go.bug.st/serial"
)

func NewSerialPrinter(portName string, baudRate uint64) (*Printer, error) {
    ports, err := serial.GetPortsList()
    if err != nil {
        return nil, fmt.Errorf("serial ports error: %w", err)
    }
    if !contains(ports, portName) {
        return nil, fmt.Errorf("port %s not found", portName)
    }

    mode := &serial.Mode{
        BaudRate: int(baudRate),
        Parity:   serial.NoParity,
        DataBits: 8,
        StopBits: serial.OneStopBit,
    }

    port, err := serial.Open(portName, mode)
    if err != nil {
        return nil, fmt.Errorf("failed to open port %s: %w", portName, err)
    }
    port.SetReadTimeout(100 * time.Millisecond)
    printer, err := NewPrinter(port)
    if err != nil {
        port.Close()
        return nil, err
    }

    port.Write([]byte{0x11})      // XON
    port.Write([]byte{0x1B, 0x40}) // ESC @
    port.Write([]byte("TEST\n"))
    printer.Init()
    return printer, nil
}

func contains(list []string, item string) bool {
    for _, s := range list {
        if s == item {
            return true
        }
    }
    return false
}
