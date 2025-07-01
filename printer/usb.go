package printer

import (
	"errors"
	"fmt"
	"io"

	"github.com/google/gousb"
)

// usbConn – обёртка для USB-подключения, реализующая io.ReadWriteCloser.
// Она инкапсулирует gousb.Context, Device, Config, Interface и endpoint’ы для записи (и чтения, если потребуется).
type usbConn struct {
	ctx  *gousb.Context
	dev  *gousb.Device
	cfg  *gousb.Config
	intf *gousb.Interface
	out  *gousb.OutEndpoint
	in   *gousb.InEndpoint // Может быть nil, если чтение не требуется
}

// nopCloser оборачивает io.ReadWriter и реализует метод Close(), который ничего не делает.
type nopCloser struct {
	io.ReadWriter
}

func (nopCloser) Close() error {
	return nil
}

// NewUSBPrinter создаёт Printer через USB, принимая vendorID и productID.
// Он настраивает USB-соединение и возвращает Printer, которому "безразлично"
// – USB это или сетевой принтер, поскольку далее используется универсальный интерфейс io.ReadWriteCloser.
func NewUSBPrinter(vendorID, productID gousb.ID) (*Printer, error) {
	// Инициализируем USB контекст.
	ctx := gousb.NewContext()
	// Включаем автодетач, чтобы libusb автоматически отключал kernel драйвер для захвата интерфейса.
	ctx.Debug(0) // Можно выставить уровень дебага, если нужно.

	// Ищем устройство с заданными идентификаторами.
	dev, err := findUSBPrinter(ctx, vendorID, productID)
	if err != nil {
		ctx.Close()
		return nil, err
	}

	err = dev.SetAutoDetach(true)
	if err != nil {
		dev.Close()
		ctx.Close()
		return nil, fmt.Errorf("failed to set auto detach: %w", err)
	}

	// Устанавливаем конфигурацию (часто используется 1).
	cfg, err := dev.Config(1)
	if err != nil {
		dev.Close()
		ctx.Close()
		return nil, err
	}

	// Выбираем интерфейс (0,0). В некоторых случаях может понадобиться другой интерфейс или альтернатива.
	intf, err := cfg.Interface(0, 0)
	if err != nil {
		cfg.Close()
		dev.Close()
		ctx.Close()
		return nil, err
	}

	// Получаем выходной endpoint для передачи RAW данных (обычно номер 1).
	outEp, err := intf.OutEndpoint(0x01) // Обычно 0x01 для принтеров.  [0x01(1,OUT) 0x82(2,IN)]
	if err != nil {
		intf.Close()
		cfg.Close()
		dev.Close()
		ctx.Close()
		return nil, err
	}

	// Пытаемся открыть входной endpoint для чтения статуса (если поддерживается).
	inEp, err := intf.InEndpoint(1)
	if err != nil {
		inEp = nil // Если чтение не поддерживается – оставляем nil.
	}

	// Собираем всё в usbConn.
	uc := &usbConn{
		ctx:  ctx,
		dev:  dev,
		cfg:  cfg,
		intf: intf,
		out:  outEp,
		in:   inEp,
	}

	// Используем существующий конструктор Printer, передавая нашу обёртку,
	// которая реализует io.ReadWriteCloser.
	printer, err := NewPrinter(uc)
	if err != nil {
		uc.Close()
		return nil, err
	}

	return printer, nil
}

// Read пытается прочитать данные через inEndpoint, если он доступен.
func (u *usbConn) Read(p []byte) (int, error) {
	if u.in != nil {
		return u.in.Read(p)
	}
	// Можно вернуть ошибку или нулевое значение, если чтение не реализовано
	return 0, errors.New("read not supported on USB connection")
}

// Close закрывает все уровни USB подключения.
func (u *usbConn) Close() error {
	// Закрываем в обратном порядке
	if u.intf != nil {
		u.intf.Close()
	}
	if u.cfg != nil {
		u.cfg.Close()
	}
	if u.dev != nil {
		u.dev.Close()
	}
	if u.ctx != nil {
		u.ctx.Close()
	}
	return nil
}

// Write отправляет данные через outEndpoint.
func (u *usbConn) Write(p []byte) (int, error) {
	return u.out.Write(p)
}

func findUSBPrinter(ctx *gousb.Context, vendorID, productID gousb.ID) (*gousb.Device, error) {
	devs, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		return desc.Vendor == vendorID && desc.Product == productID
	})
	if err != nil {
		return nil, err
	}
	if len(devs) == 0 {
		return nil, fmt.Errorf("USB device %s:%s not found", vendorID, productID)
	}
	return devs[0], nil // Возвращаем первое найденное устройство
}
