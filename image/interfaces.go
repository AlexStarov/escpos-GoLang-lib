package image

// Target — интерфейс, который должен реализовывать приёмник растровых данных
type Target interface {
	Raster(width, height, bytesWidth int, rasterData []byte, printingType string)
}
