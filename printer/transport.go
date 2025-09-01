package printer

import (
    "bytes"
    "fmt"
    "io"
    "net"
    "os"
    "strings"
    "sync"
    "time"
)

type Transport interface {
    Write([]byte) (int, error)
    Read([]byte) (int, error)
    Close() error
}

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
    if err := l.flushJob(); err != nil {
        _ = l.conn.Close()
        l.closed = true
        return err
    }
    err := l.conn.Close()
    l.closed = true
    return err
}

func (l *LPDTransport) flushJob() error {
    if l.jobBuf.Len() == 0 {
        return nil
    }

    queue := l.queue
    if queue == "" {
        queue = "raw"
    }

    host, _ := os.Hostname()
    if host == "" {
        host = "localhost"
    }
    user := os.Getenv("USER")
    if user == "" {
        user = "golang"
    }

    jobID := int(time.Now().UnixNano() % 1000000)
    hostShort := host
    if i := strings.IndexByte(hostShort, '.'); i > 0 {
        hostShort = hostShort[:i]
    }
    jobName := fmt.Sprintf("escpos-%d", jobID)
    cfName := fmt.Sprintf("cfA%03d%s", jobID%1000, hostShort)
    dfName := fmt.Sprintf("dfA%03d%s", jobID%1000, hostShort)

    control := fmt.Sprintf(
        "H%s\nP%s\nJ%s\nN%s\nl%s\n",
        host, user, jobName, dfName, dfName,
    )

    if err := l.writeAll([]byte{0x02}); err != nil {
        return err
    }
    if err := l.writeAll([]byte(queue + "\n")); err != nil {
        return err
    }
    if err := l.expectACK(); err != nil {
        return fmt.Errorf("LPD no ACK on initial job start: %w", err)
    }

    if err := l.writeAll([]byte{0x02}); err != nil {
        return err
    }
    ctrlHeader := fmt.Sprintf("%d %s\n", len(control), cfName)
    if err := l.writeAll([]byte(ctrlHeader)); err != nil {
        return err
    }
    if err := l.writeAll([]byte(control)); err != nil {
        return err
    }
    if err := l.writeAll([]byte{0x00}); err != nil {
        return err
    }
    if err := l.expectACK(); err != nil {
        return fmt.Errorf("LPD no ACK after control file: %w", err)
    }

    data := l.jobBuf.Bytes()
    if err := l.writeAll([]byte{0x03}); err != nil {
        return err
    }
    dataHeader := fmt.Sprintf("%d %s\n", len(data), dfName)
    if err := l.writeAll([]byte(dataHeader)); err != nil {
        return err
    }
    if err := l.writeAll(data); err != nil {
        return err
    }
    if err := l.writeAll([]byte{0x00}); err != nil {
        return err
    }
    if err := l.expectACK(); err != nil {
        return fmt.Errorf("LPD no ACK after data file: %w", err)
    }

    l.jobBuf.Reset()
    return nil
}

func (l *LPDTransport) writeAll(b []byte) error {
    total := 0
    for total < len(b) {
        n, err := l.conn.Write(b[total:])
        if err != nil {
            return err
        }
        total += n
    }
    return nil
}

func (l *LPDTransport) expectACK() error {
    _ = l.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
    defer l.conn.SetReadDeadline(time.Time{})
    buf := []byte{0}
    n, err := l.conn.Read(buf)
    if err != nil {
        return err
    }
    if n != 1 || buf[0] != 0x00 {
        return fmt.Errorf("unexpected ACK byte: %v (n=%d)", buf[0], n)
    }
    return nil
}

type nopCloser struct {
    io.ReadWriter
}

func (n nopCloser) Close() error { return nil }
