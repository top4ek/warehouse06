// Package gopher provides the read-only Gopher TCP interface to the catalog.
package gopher

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"go.uber.org/zap"

	"warehouse06/internal/repository"
)

const transferIdleTimeout = 60 * time.Second

type Config struct {
	Host       string
	Port       int
	StorageDir string
}

type Server struct {
	cfg       Config
	holder    *repository.Holder
	log       *zap.Logger
	mu        sync.Mutex
	listener  net.Listener
	conns     map[net.Conn]struct{}
	closed    bool
	wg        sync.WaitGroup
	queries   map[string]string
	queryKeys []string
}

func New(cfg Config, holder *repository.Holder, log *zap.Logger) (*Server, error) {
	if cfg.Host == "" || len(cfg.Host) > 100 || strings.ContainsAny(cfg.Host, " \t\r\n/:?") || !utf8.ValidString(cfg.Host) {
		return nil, errors.New("invalid Gopher public host")
	}
	for _, r := range cfg.Host {
		if unicode.IsControl(r) {
			return nil, errors.New("invalid Gopher public host")
		}
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return nil, errors.New("invalid Gopher public port")
	}
	if holder == nil || log == nil {
		return nil, errors.New("gopher repository and logger are required")
	}
	return &Server{cfg: cfg, holder: holder, log: log, conns: make(map[net.Conn]struct{}), queries: make(map[string]string)}, nil
}

// Serve accepts one request per connection. Close interrupts active transfers.
func (s *Server) Serve(listener net.Listener) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		_ = listener.Close()
		return nil
	}
	if s.listener != nil {
		s.mu.Unlock()
		return errors.New("gopher server is already serving")
	}
	s.listener = listener
	s.mu.Unlock()
	for {
		conn, err := listener.Accept()
		if err != nil {
			s.mu.Lock()
			closed := s.closed
			s.mu.Unlock()
			if closed {
				return nil
			}
			return err
		}
		s.mu.Lock()
		if s.closed {
			s.mu.Unlock()
			_ = conn.Close()
			return nil
		}
		if len(s.conns) >= 128 {
			s.mu.Unlock()
			_ = conn.Close()
			continue
		}
		s.conns[conn] = struct{}{}
		s.wg.Add(1)
		s.mu.Unlock()
		go func() {
			defer s.wg.Done()
			defer func() { _ = conn.Close(); s.mu.Lock(); delete(s.conns, conn); s.mu.Unlock() }()
			if err := conn.SetReadDeadline(time.Now().Add(transferIdleTimeout)); err != nil {
				return
			}
			s.handle(conn)
		}()
	}
}

func (s *Server) Close() error {
	s.mu.Lock()
	s.closed = true
	if s.listener != nil {
		_ = s.listener.Close()
	}
	for conn := range s.conns {
		_ = conn.Close()
	}
	s.mu.Unlock()
	s.wg.Wait()
	return nil
}

func (s *Server) handle(conn net.Conn) {
	w := &responseWriter{conn: conn}
	line, err := bufio.NewReaderSize(conn, 4096).ReadSlice('\n')
	if err != nil || !strings.HasSuffix(string(line), "\r\n") || !utf8.Valid(line) {
		s.writeError(w, "Invalid request")
		return
	}
	request := string(line[:len(line)-2])
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := s.respond(ctx, w, request); err != nil {
		s.log.Debug("Gopher request failed", zap.Error(err))
		if !w.started {
			message := "Not found or invalid request"
			if errors.Is(err, errSearchExpired) {
				message = "Search expired; enter query again"
			}
			s.writeError(w, message)
		}
	}
}

// A failed transfer must end at EOF, without appending an error menu to a file.
type responseWriter struct {
	conn    net.Conn
	started bool
}

func (w *responseWriter) Write(p []byte) (int, error) {
	written := 0
	for len(p) > 0 {
		// Refresh the inactivity window on each bounded write instead of
		// imposing a total lifetime on healthy slow file transfers.
		if err := w.conn.SetWriteDeadline(time.Now().Add(transferIdleTimeout)); err != nil {
			return written, err
		}
		chunk := p[:min(len(p), 512)]
		n, err := w.conn.Write(chunk)
		written += n
		if n > 0 {
			w.started = true
		}
		if err != nil {
			return written, err
		}
		if n != len(chunk) {
			return written, io.ErrShortWrite
		}
		p = p[n:]
	}
	return written, nil
}

func (s *Server) line(kind byte, name, selector string) string {
	return fmt.Sprintf("%c%s\t%s\t%s\t%d\r\n", kind, name, selector, s.cfg.Host, s.cfg.Port)
}

func (s *Server) writeError(w io.Writer, message string) {
	_, _ = io.WriteString(w, s.line('3', message, "")+".\r\n")
}
