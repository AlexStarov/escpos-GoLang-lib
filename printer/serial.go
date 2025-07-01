package printer

import (
	"fmt"
	"log"
	"time"

	"go.bug.st/serial"

	logInternal "github.com/AlexStarov/escpos-GoLang-lib/log" // Убедитесь, что пакет image импортирован корректно
)

// NewSerialPrinter создаёт Printer через последовательный порт (COM или /dev/cu.usbmodem*).
func NewSerialPrinter(portName string, baudRate uint64) (*Printer, error) {
	// Получаем список доступных портов (можно использовать для проверки)
	ports, err := serial.GetPortsList()
	if err != nil {
		logInternal.Errlog.Printf("Ошибка получения списка портов: %v", err)
		return nil, fmt.Errorf("failed to list serial ports: %w", err)
	}
	logInternal.Stdlog.Printf("Доступные порты: %v", ports)

	// Проверяем, существует ли заданный порт
	if !contains(ports, portName) {
		log.Printf("Порт %s не найден", portName)
		return nil, fmt.Errorf("serial port %s not found", portName)
	}

	// Открываем порт с указанной скоростью
	mode := &serial.Mode{
		BaudRate: int(baudRate),
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}

	logInternal.Stdlog.Printf("Пытаемся открыть порт: %s с baudRate: %d", portName, baudRate)

	serialPort, err := serial.Open(portName, mode)

	logInternal.Stdlog.Printf("Отладка: OpenPort завершился") // <- Отладочный лог

	if err != nil {
		log.Printf("Ошибка открытия порта %s: %v", portName, err)
		return nil, fmt.Errorf("failed to open serial port %s: %w", portName, err)
	}
	logInternal.Stdlog.Printf("Порт %s успешно открыт", portName)
	logInternal.Stdlog.Println(serialPort)
	serialPort.SetReadTimeout(100 * time.Millisecond) // Устанавливаем таймаут чтения

	// Создаём объект Printer.
	printer, err := NewPrinter(serialPort)
	if err != nil {
		serialPort.Close()
		return nil, err
	}

	n, err := serialPort.Write([]byte{0x11}) // XON
	logInternal.Stdlog.Printf("Отправлено %d байт на принтер: %x", n, []byte{0x11})

	if err != nil {
		log.Printf("Ошибка отправки команды инициализации: %v", err)
	}

	_, err = serialPort.Write([]byte{0x1B, 0x40}) // ESC @ (инициализация)
	if err != nil {
		log.Printf("Ошибка отправки команды инициализации: %v", err)
	}
	logInternal.Stdlog.Printf("Отправлено %d байт на принтер: %x", n, []byte{0x1B, 0x40})

	_, err = serialPort.Write([]byte{0x1B, 0x21, 0x00}) // Установка обычного текста
	_, err = serialPort.Write([]byte("TEST PRINT\n"))
	// serialPort.Write([]byte("01234567890\n")) // Отправляем строку для инициализации

	printer.Init()
	printer.Reset()
	printer.Write([]byte("1234567890\n")) // ESC @ (Initialize printer)
	time.Sleep(100 * time.Millisecond)    // Даем время на инициализацию
	return printer, nil
}

// Проверяем, есть ли порт в списке
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
