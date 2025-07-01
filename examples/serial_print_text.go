package main

import (
	"log"

	"github.com/AlexStarov/escpos-GoLang-lib/printer"
)

func PrintToSerial() {
	p, err := printer.NewSerialPrinter("COM3", 9600)
	if err != nil {
		log.Fatal("Serial error: ", err)
	}
	p.Init()
	p.Write([]byte("Проверка связи\n"))
	p.Cash()
}
