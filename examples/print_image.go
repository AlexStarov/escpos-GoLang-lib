package main

import (
	"log"

	"github.com/AlexStarov/escpos-GoLang-lib/printer"
)

func PrintImage() {
	p, err := printer.NewSerialPrinter("/dev/ttyUSB0", 115200)
	if err != nil {
		log.Fatal(err)
	}
	if err := p.PrintImage("logo.png"); err != nil {
		log.Println("Ошибка изображения:", err)
	}
	p.Linefeed()
	p.Cut()
}
