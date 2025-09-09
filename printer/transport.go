package printer

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Transport interface {
	Write([]byte) (int, error)
	Read([]byte) (int, error)
	Close() error
}

// -------------------- RAW --------------------

type RawTransport struct {
	conn io.ReadWriteCloser
}

func (r *RawTransport) Write(b []byte) (int, error) { return r.conn.Write(b) }
func (r *RawTransport) Read(b []byte) (int, error)  { return r.conn.Read(b) }
func (r *RawTransport) Close() error                { return r.conn.Close() }

type LPDTransport struct {
	conn   net.Conn
	queue  string
	jobBuf bytes.Buffer
	closed bool
	mu     sync.Mutex
}

func NewLPDTransport(conn net.Conn, queue string) *LPDTransport {
	if queue == "" {
		queue = "lp"
	}
	return &LPDTransport{
		conn:  conn,
		queue: queue,
	}
}

func (l *LPDTransport) Write(data []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return 0, io.ErrClosedPipe
	}
	return l.jobBuf.Write(data)
}

func (l *LPDTransport) Read(b []byte) (int, error) {
	return l.conn.Read(b)
}

func (l *LPDTransport) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return nil
	}
	defer func() { l.closed = true }()

	log.Printf("[DEBUG] LPDTransport.Close(): jobBuf.Len()=%d", l.jobBuf.Len())

	if l.jobBuf.Len() == 0 {
		log.Printf("[DEBUG] jobBuf пуст — просто закрываем соединение")
		return l.conn.Close()
	}

	log.Printf("[DEBUG] Отправляем задание в LPD...")
	if err := l.flushJob(); err != nil {
		log.Printf("[DEBUG] flushJob() вернул ошибку: %v", err)
		_ = l.conn.Close()
		return err
	}

	log.Printf("[DEBUG] Закрываем соединение после flushJob()")
	return l.conn.Close()
}

func (l *LPDTransport) flushJob() error {
	host, _ := os.Hostname()
	if host == "" {
		host = "localhost"
	}
	user := os.Getenv("USER")
	if user == "" {
		user = "GoLang"
	}

	jobID := int(time.Now().UnixNano() % 1000000)
	hostShort := host
	if i := strings.IndexByte(hostShort, '.'); i > 0 {
		hostShort = hostShort[:i]
	}
	jobName := fmt.Sprintf("escpos-%d", jobID)
	cfName := fmt.Sprintf("cfA%03d%s", jobID%1000, hostShort)
	dfName := fmt.Sprintf("dfA%03d%s", jobID%1000, hostShort)

	// Минимально корректный control file
	// H - host, P - user, J - job name, N - original file name, U - data file to print
	control := fmt.Sprintf(
		"H%s\nP%s\nJ%s\nN%s\nU%s\n",
		host, user, jobName, dfName, dfName,
	)

	log.Printf("[DEBUG] Stage 1: requestPrintJob")
	if err := requestPrintJob(l.conn, l.queue); err != nil {
		return fmt.Errorf("LPD: stage 1 failed: %w", err)
	}

	log.Printf("[DEBUG] Stage 2: sendControlFile")
	if err := sendControlFile(l.conn, l.queue, cfName, []byte(control)); err != nil {
		return fmt.Errorf("LPD: stage 2 failed: %w", err)
	}

	// Этап 3: файл данных (точный размер из буфера)
	data := l.jobBuf.Bytes()
	log.Printf("[DEBUG] Stage 3: sendDataFile (len=%d)", len(data))
	if err := sendDataFile(l.conn, l.queue, dfName, data); err != nil {
		return fmt.Errorf("LPD: stage 3 failed: %w", err)
	}

	log.Printf("[DEBUG] All stages LPD finished successfully")

	// Успех — очистим буфер
	l.jobBuf.Reset()
	return nil
}

// -------------------- LPD helpers --------------------

func requestPrintJob(conn net.Conn, queue string) error {
	// \x02 + <queue>\n
	log.Printf("[DEBUG] requestPrintJob: conn=%v", conn)
	log.Printf("[DEBUG] requestPrintJob: queue=%q", queue)
	if err := writeAll(conn, []byte{0x02}); err != nil {
		return err
	}
	if err := writeAll(conn, []byte(queue+"\n")); err != nil {
		return err
	}
	return readAck(conn, "stage 1")
}

func sendControlFile(conn net.Conn, queue, cfName string, control []byte) error {
	// \x02 + "<size> <cfName>\n" + <control> + \x00
	if err := writeAll(conn, []byte{0x02}); err != nil {
		return err
	}
	header := []byte(strconv.Itoa(len(control)) + " " + cfName + "\n")
	if err := writeAll(conn, header); err != nil {
		return err
	}
	if err := writeAll(conn, control); err != nil {
		return err
	}
	if err := writeAll(conn, []byte{0x00}); err != nil {
		return err
	}
	return readAck(conn, "stage 2")
}

func sendDataFile(conn net.Conn, queue, dfName string, data []byte) error {
	// \x03 + "<size> <dfName>\n" + <data> + \x00
	if err := writeAll(conn, []byte{0x03}); err != nil {
		return err
	}
	header := []byte(strconv.Itoa(len(data)) + " " + dfName + "\n")
	if err := writeAll(conn, header); err != nil {
		return err
	}
	if err := writeAll(conn, data); err != nil {
		return err
	}
	if err := writeAll(conn, []byte{0x00}); err != nil {
		return err
	}
	return readAck(conn, "stage 3")
}

func readAck(conn net.Conn, stage string) error {
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	defer conn.SetReadDeadline(time.Time{})

	ack := make([]byte, 1)
	log.Printf("[DEBUG] Waiting for ACK (%s)...", stage)
	n, err := conn.Read(ack)
	if err != nil {
		return fmt.Errorf("ERROR reading ACK on %s: %w", stage, err)
	}
	log.Printf("[DEBUG] Received ACK byte: 0x%02x (n=%d) on %s", ack[0], n, stage)
	if n != 1 || ack[0] != 0x00 {
		return fmt.Errorf("LPD request not acknowledged on %s", stage)
	}
	return nil
}

func writeAll(conn net.Conn, b []byte) error {
	sent := 0
	for sent < len(b) {
		n, err := conn.Write(b[sent:])
		if err != nil {
			return err
		}
		sent += n
	}
	return nil
}

// -------------------- helpers --------------------

type nopCloser struct {
	io.ReadWriter
}

func (n nopCloser) Close() error { return nil }
