package main

import (
	"github.com/AlexStarov/escpos-GoLang-lib/printer"
)

func sendRawCommands(p *printer.Printer) {
	p.Write([]byte("\x1B\x61\x01")) // Выравнивание по центру
	p.Write([]byte("Заголовок\n"))
	p.Write([]byte("\x1B\x45\x01")) // Жирный
	p.Write([]byte("Внимание!\n"))
	p.Write([]byte("\x1B\x45\x00")) // Обычный
}
