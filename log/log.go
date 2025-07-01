package printer

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

// Определяем уровни логирования
const (
	DEBUG = "DEBUG"
	INFO  = "INFO"
	WARN  = "WARN"
	ERROR = "ERROR"
)

var Stdlog, Errlog *log.Logger

func init() {
	Stdlog = log.New(os.Stdout, "Success: ", log.Ldate|log.Ltime)
	Errlog = log.New(os.Stderr, "Error: ", log.Ldate|log.Ltime)
}

func LogMessage(level, message string) {

	// Получаем имя лог-файла и текущий суффикс.
	logPath, suffix := getLogFilePath("stdlog")

	// Производим ротацию: если используется log/app-2.log, удаляем log/app-0.log.
	rotateLogs(suffix)

	// Открываем файл для логирования с режимами добавления, создания и записи.
	logFile, localErr := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if localErr != nil {
		Errlog.Fatalf("[ERROR] Ошибка открытия файла %s: %v", logPath, localErr)
	}
	defer logFile.Close()

	// Создаем multi-writer и новый экземпляр логгера.
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	logger := log.New(multiWriter, "Success: ", log.Ldate|log.Ltime)

	if level == ERROR {
		err := errors.New(message)
		PrintIfErr("", &err)
	}

	logger.Printf("[%s] %s\n", level, message)
}

// getLogFilePath определяет имя файла лога и возвращает суффикс для ротации.
func getLogFilePath(typeLog string) (string, int) {
	now := time.Now()
	day := now.Day()
	var suffix int
	switch {
	case day >= 1 && day <= 9:
		suffix = 0
	case day >= 10 && day <= 19:
		suffix = 1
	default: // day >= 20
		suffix = 2
	}
	// Имя файла формируется с использованием суффикса.
	filepath := fmt.Sprintf("log/%s-%d.log", typeLog, suffix)
	return filepath, suffix
}

// rotateLogs реализует логику "круговой" ротации.
// Если текущий лог находится в файле с суффиксом 2 (log/app-2.log),
// то удаляет лог с суффиксом 0.
func rotateLogs(currentSuffix int) {
	var fileToDelete string

	// Определяем, какой файл нужно удалить, исходя из текущего суффикса.
	switch currentSuffix {
	case 2:
		fileToDelete = "log/app-0.log"
	case 1:
		fileToDelete = "log/app-2.log"
	case 0:
		fileToDelete = "log/app-1.log"
	default:
		// Если currentSuffix не 0, 1 или 2, выходим.
		return
	}

	// Если файл для удаления существует, пытаемся его удалить.
	if _, err := os.Stat(fileToDelete); err == nil {
		if err := os.Remove(fileToDelete); err != nil {
			log.Printf("[WARN] Ошибка при удалении файла %s: %v", fileToDelete, err)
		} else {
			log.Printf("[INFO] Файл %s успешно удален", fileToDelete)
		}
	}
}

func PrintIfErr(msg string, err *error) {

	if *err != nil {
		// Получаем имя лог-файла и текущий суффикс.
		logPath, suffix := getLogFilePath("errors")

		// Производим ротацию: если используется log/app-2.log, удаляем log/app-0.log.
		rotateLogs(suffix)

		// Открываем файл для логирования с режимами добавления, создания и записи.
		logFile, localErr := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if localErr != nil {
			Errlog.Fatalf("[ERROR] Ошибка открытия файла %s: %v", logPath, localErr)
		}
		defer logFile.Close()

		// Создаем multi-writer и новый экземпляр логгера.
		multiWriter := io.MultiWriter(os.Stdout, logFile)
		logger := log.New(multiWriter, "Error: ", log.Ldate|log.Ltime)

		logger.Printf("%s: %v\n", msg, *err)
	}
}
