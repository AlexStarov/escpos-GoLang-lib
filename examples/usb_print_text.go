package main

import (
	"log"

	"github.com/AlexStarov/escpos-GoLang-lib/printer"
)

func PrintToUSB() {
	p, err := printer.NewUSBPrinter(0x1234, 0x5678)
	if err != nil {
		log.Fatal("USB error: ", err)
	}

	p.Init()
	p.SetAlign("center")
	p.SetFontSize(2, 2)
	p.Write([]byte("Добро пожаловать!\n"))
	p.Cut()
}
