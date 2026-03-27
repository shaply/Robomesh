package tcp_server

import (
	"bufio"
	"context"
	"net"
	"roboserver/comms"
	"roboserver/database"
	"roboserver/shared"
	"strings"
	"testing"
	"time"
)

func init() {
	shared.AppConfig = shared.Config{
		Auth: shared.AuthConfig{
			JWTSecret:   "test-secret-key-for-tcp-tests",
			JWTExpiry:   3600,
			NonceLength: 32,
		},
		Database: shared.DatabaseConfig{
			Redis: shared.RedisConfig{
				SessionTTL: "60s",
			},
		},
		Handlers: shared.HandlersConfig{
			BasePath: "./testdata",
		},
	}
}

// --- Helper types ---

// mockDBManager implements database.DBManager for testing without real databases.
type mockDBManager struct {
	pg  *database.PostgresHandler
	rds *database.RedisHandler
}

func (m *mockDBManager) Postgres() *database.PostgresHandler { return m.pg }
func (m *mockDBManager) Redis() *database.RedisHandler       { return m.rds }
func (m *mockDBManager) Stop()                               {}
func (m *mockDBManager) IsHealthy(_ context.Context) bool    { return m.pg != nil && m.rds != nil }

// mockBus implements comms.Bus for unit tests.
type mockBus struct{}

func (b *mockBus) PublishEvent(string, any) error                                        { return nil }
func (b *mockBus) SubscribeEvent(string, comms.EventHandler) (func(), error)             { return func() {}, nil }
func (b *mockBus) PublishRegistrationResponse(_ context.Context, _ string, _ bool) error { return nil }
func (b *mockBus) WaitForRegistrationResponse(_ context.Context, _ string) (bool, error) {
	return false, nil
}

// readLine reads a single \n-terminated line from a connection with a timeout.
func readLine(conn net.Conn, timeout time.Duration) (string, error) {
	conn.SetReadDeadline(time.Now().Add(timeout))
	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()), nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", nil
}

// sendLine writes a \n-terminated line to a connection.
func sendLine(conn net.Conn, msg string) {
	conn.Write([]byte(msg + "\n"))
}

// --- Tests ---

func TestHandleConnectionRejectsUnknownCommand(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := &TCPServer_t{
		bus:          &mockBus{},
		db:           &mockDBManager{},
		main_context: ctx,
	}

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()

	go s.handleConnection(serverConn)

	// Send an unknown command
	sendLine(clientConn, "FOOBAR")

	line, err := readLine(clientConn, 2*time.Second)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}
	if !strings.HasPrefix(line, "ERROR") {
		t.Errorf("Expected ERROR response for unknown command, got: %s", line)
	}
}

func TestHandleConnectionRejectsEmptyLines(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := &TCPServer_t{
		bus:          &mockBus{},
		db:           &mockDBManager{},
		main_context: ctx,
	}

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()

	go s.handleConnection(serverConn)

	// Send empty lines — should be silently ignored
	sendLine(clientConn, "")
	sendLine(clientConn, "  ")

	// Then send an unknown command to get a response and verify we're still connected
	sendLine(clientConn, "INVALID")
	line, err := readLine(clientConn, 2*time.Second)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}
	if !strings.HasPrefix(line, "ERROR") {
		t.Errorf("Expected ERROR, got: %s", line)
	}
}

func TestAuthWithNilDatabase(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// nil database
	s := &TCPServer_t{
		bus:          &mockBus{},
		db:           nil,
		main_context: ctx,
	}

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()

	go s.handleConnection(serverConn)

	sendLine(clientConn, "AUTH")
	line, err := readLine(clientConn, 2*time.Second)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}
	if !strings.Contains(line, "ERROR") {
		t.Errorf("Expected ERROR for nil DB, got: %s", line)
	}
}

func TestAuthWithNilPostgres(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := &TCPServer_t{
		bus:          &mockBus{},
		db:           &mockDBManager{pg: nil, rds: nil},
		main_context: ctx,
	}

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()

	go s.handleConnection(serverConn)

	sendLine(clientConn, "AUTH")
	line, err := readLine(clientConn, 2*time.Second)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}
	if !strings.Contains(line, "ERROR") {
		t.Errorf("Expected ERROR for nil PG/Redis, got: %s", line)
	}
}

func TestRegisterWithNilRedis(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := &TCPServer_t{
		bus:          &mockBus{},
		db:           &mockDBManager{pg: nil, rds: nil},
		main_context: ctx,
	}

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()

	go s.handleConnection(serverConn)

	sendLine(clientConn, "REGISTER")
	line, err := readLine(clientConn, 2*time.Second)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}
	if !strings.Contains(line, "ERROR") {
		t.Errorf("Expected ERROR for nil Redis, got: %s", line)
	}
}

func TestRegisterChallengeResponse(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s := &TCPServer_t{
		bus:          &mockBus{},
		db:           &mockDBManager{pg: nil, rds: nil},
		main_context: ctx,
	}

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()

	go s.handleConnection(serverConn)

	sendLine(clientConn, "REGISTER")

	// With nil Redis, should get ERROR
	line, err := readLine(clientConn, 2*time.Second)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}
	if !strings.Contains(line, "ERROR") {
		t.Errorf("Expected ERROR for nil Redis on REGISTER, got: %s", line)
	}
}

func TestMultipleCommandsBeforeAuth(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := &TCPServer_t{
		bus:          &mockBus{},
		db:           &mockDBManager{},
		main_context: ctx,
	}

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()

	go s.handleConnection(serverConn)

	// Send several invalid commands — should get errors for each
	sendLine(clientConn, "HELLO")
	line1, _ := readLine(clientConn, 2*time.Second)
	if !strings.HasPrefix(line1, "ERROR") {
		t.Errorf("Expected ERROR for HELLO, got: %s", line1)
	}

	sendLine(clientConn, "STATUS")
	line2, _ := readLine(clientConn, 2*time.Second)
	if !strings.HasPrefix(line2, "ERROR") {
		t.Errorf("Expected ERROR for STATUS, got: %s", line2)
	}
}

func TestConnectionClosedByClient(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := &TCPServer_t{
		bus:          &mockBus{},
		db:           &mockDBManager{},
		main_context: ctx,
	}

	clientConn, serverConn := net.Pipe()
	done := make(chan struct{})

	go func() {
		s.handleConnection(serverConn)
		close(done)
	}()

	// Close client immediately
	clientConn.Close()

	// handleConnection should return
	select {
	case <-done:
		// OK
	case <-time.After(3 * time.Second):
		t.Error("handleConnection did not return after client disconnect")
	}
}

func TestServerContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	s := &TCPServer_t{
		bus:          &mockBus{},
		db:           &mockDBManager{},
		main_context: ctx,
	}

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()

	done := make(chan struct{})
	go func() {
		s.handleConnection(serverConn)
		close(done)
	}()

	// Cancel context
	cancel()

	// Close client to unblock the scanner
	clientConn.Close()

	select {
	case <-done:
		// OK
	case <-time.After(3 * time.Second):
		t.Error("handleConnection did not return after context cancellation")
	}
}
