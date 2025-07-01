# PrinterLib

Библиотека для печати по ESC/POS через USB и Serial порты.

## Установка

```bash
go get github.com/AlexStarov/printerlib

## Дерево проекта
printerlib/
├── go.mod
├── README.md
│
├── printer/
│   ├── printer.go
│   ├── usb.go
│   ├── serial.go
│
├── image/
│   ├── convert.go
│   ├── raster.go
│
├── util/
│   └── escpos.go
│
├── examples/
│   ├── usb_print_text.go
│   ├── serial_print_text.go
│   ├── print_image.go
│   ├── escpos_manual.go
│
├── docs/
│   ├── usb.md
│   ├── serial.md
│   ├── image.md
│   ├── escpos.md
