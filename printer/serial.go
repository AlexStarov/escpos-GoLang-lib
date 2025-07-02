package printer

import (
    "fmt"
    "time"

    "go.bug.st/serial"

    logInternal "github.com/AlexStarov/escpos-GoLang-lib/log"
)

// NewSerialPrinter создаёт и инициализирует ESC/POS-принтер через последовательный порт.
func NewSerialPrinter(portName string, baudRate uint64) (*Printer, error) {
    // Получаем список доступных COM-портов
    ports, err := serial.GetPortsList()
    if err != nil {
        logInternal.Errlog.Printf("Ошибка получения списка портов: %v", err)
        return nil, fmt.Errorf("failed to list serial ports: %w", err)
    }
    logInternal.Stdlog.Printf("Доступные порты: %v", ports)

    // Проверяем, что указанный порт существует
    if !contains(ports, portName) {
        logInternal.Errlog.Printf("Порт %s не найден", portName)
        return nil, fmt.Errorf("serial port %s not found", portName)
    }
    logInternal.Stdlog.Printf("Используем порт: %s", portName)

    // Конфигурируем параметры подключения
    mode := &serial.Mode{
        BaudRate: int(baudRate),
        DataBits: 8,
        Parity:   serial.NoParity,
        StopBits: serial.OneStopBit,
    }

    // Открываем порт
    serialPort, err := serial.Open(portName, mode)
    if err != nil {
        logInternal.Errlog.Printf("Не удалось открыть порт %s: %v", portName, err)
        return nil, fmt.Errorf("failed to open serial port %s: %w", portName, err)
    }
    logInternal.Stdlog.Printf("Порт %s успешно открыт", portName)

    // Устанавливаем таймаут чтения
    serialPort.SetReadTimeout(100 * time.Millisecond)

    // Создаём объект Printer
    printer, err := NewPrinter(serialPort)
    if err != nil {
        serialPort.Close()
        logInternal.Errlog.Printf("Ошибка инициализации Printer: %v", err)
        return nil, err
    }

    // Отправляем XON для разблокировки приёма
    if n, err := serialPort.Write([]byte{0x11}); err != nil {
        logInternal.Errlog.Printf("Не удалось отправить XON: %v", err)
    } else {
        logInternal.Stdlog.Printf("Отправлено XON (%d байт)", n)
    }

    // ESC @ — полная инициализация принтера
    if n, err := serialPort.Write([]byte{0x1B, 0x40}); err != nil {
        logInternal.Errlog.Printf("Не удалось отправить ESC @: %v", err)
    } else {
        logInternal.Stdlog.Printf("Отправлено ESC @ (%d байт)", n)
    }

    // Даем принтеру время на завершение инициализации
    time.Sleep(100 * time.Millisecond)

    return printer, nil
}

// contains проверяет наличие элемента item в срезе list
func contains(list []string, item string) bool {
    for _, entry := range list {
        if entry == item {
            return true
        }
    }
    return false
}
