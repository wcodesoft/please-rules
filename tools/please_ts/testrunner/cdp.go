package testrunner

import (
	"bufio"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// CDPClient manages a WebSocket connection to Chrome DevTools Protocol.
type CDPClient struct {
	conn      net.Conn
	reader    *bufio.Reader
	writeMu   sync.Mutex
	msgID     int64
	pendingMu sync.Mutex
	pending   map[int64]chan cdpResponse
	closed    int32
}

type cdpResponse struct {
	Result json.RawMessage
	Error  error
}

// DialCDP establishes a WebSocket connection to the given DevTools target URL.
func DialCDP(targetURL string) (*CDPClient, error) {
	addr := strings.TrimPrefix(targetURL, "ws://")
	slashIdx := strings.Index(addr, "/")
	if slashIdx == -1 {
		return nil, fmt.Errorf("invalid CDP websocket URL: %s", targetURL)
	}
	host := addr[:slashIdx]
	path := addr[slashIdx:]

	conn, err := net.DialTimeout("tcp", host, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed connecting to CDP at %s: %w", host, err)
	}

	key := "dGhlIHNhbXBsZSBub25jZQ=="
	req := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\n\r\n", path, host, key)
	if _, err := conn.Write([]byte(req)); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed sending websocket upgrade: %w", err)
	}

	reader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(reader, nil)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("websocket handshake failed: %w", err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		conn.Close()
		return nil, fmt.Errorf("unexpected handshake status: %d", resp.StatusCode)
	}

	client := &CDPClient{
		conn:    conn,
		reader:  reader,
		pending: make(map[int64]chan cdpResponse),
	}

	go client.readLoop()
	return client, nil
}

func (c *CDPClient) readLoop() {
	for {
		header := make([]byte, 2)
		if _, err := io.ReadFull(c.reader, header); err != nil {
			c.failAll(err)
			return
		}

		opcode := header[0] & 0x0F
		// Connection close opcode
		if opcode == 0x08 {
			c.failAll(io.EOF)
			return
		}

		rawLen := int(header[1] & 0x7F)
		payloadLen := rawLen
		if rawLen == 126 {
			var b [2]byte
			if _, err := io.ReadFull(c.reader, b[:]); err != nil {
				c.failAll(err)
				return
			}
			payloadLen = int(binary.BigEndian.Uint16(b[:]))
		} else if rawLen == 127 {
			var b [8]byte
			if _, err := io.ReadFull(c.reader, b[:]); err != nil {
				c.failAll(err)
				return
			}
			payloadLen = int(binary.BigEndian.Uint64(b[:]))
		}

		payload := make([]byte, payloadLen)
		if _, err := io.ReadFull(c.reader, payload); err != nil {
			c.failAll(err)
			return
		}

		var msg struct {
			ID     int64           `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Message string `json:"message"`
				Code    int    `json:"code"`
			} `json:"error"`
		}

		if err := json.Unmarshal(payload, &msg); err == nil && msg.ID != 0 {
			c.pendingMu.Lock()
			ch, ok := c.pending[msg.ID]
			if ok {
				delete(c.pending, msg.ID)
			}
			c.pendingMu.Unlock()

			if ok {
				if msg.Error != nil {
					ch <- cdpResponse{Error: fmt.Errorf("CDP error %d: %s", msg.Error.Code, msg.Error.Message)}
				} else {
					ch <- cdpResponse{Result: msg.Result}
				}
			}
		}
	}
}

func (c *CDPClient) failAll(err error) {
	c.pendingMu.Lock()
	defer c.pendingMu.Unlock()
	for id, ch := range c.pending {
		ch <- cdpResponse{Error: err}
		delete(c.pending, id)
	}
}

// Send sends a JSON-RPC command over CDP and awaits the response.
func (c *CDPClient) Send(method string, params map[string]interface{}) (json.RawMessage, error) {
	if atomic.LoadInt32(&c.closed) == 1 {
		return nil, fmt.Errorf("cdp client is closed")
	}

	id := atomic.AddInt64(&c.msgID, 1)
	reqObj := map[string]interface{}{
		"id":     id,
		"method": method,
	}
	if params != nil {
		reqObj["params"] = params
	}

	payload, err := json.Marshal(reqObj)
	if err != nil {
		return nil, err
	}

	ch := make(chan cdpResponse, 1)
	c.pendingMu.Lock()
	c.pending[id] = ch
	c.pendingMu.Unlock()

	// Build masked RFC 6455 frame
	frame := []byte{0x81} // FIN + text opcode
	pLen := len(payload)
	var mask [4]byte
	_, _ = rand.Read(mask[:])

	if pLen < 126 {
		frame = append(frame, byte(pLen|0x80))
	} else if pLen <= 65535 {
		frame = append(frame, 126|0x80)
		var b [2]byte
		binary.BigEndian.PutUint16(b[:], uint16(pLen))
		frame = append(frame, b[:]...)
	} else {
		frame = append(frame, 127|0x80)
		var b [8]byte
		binary.BigEndian.PutUint64(b[:], uint64(pLen))
		frame = append(frame, b[:]...)
	}
	frame = append(frame, mask[:]...)
	for i, b := range payload {
		frame = append(frame, b^mask[i%4])
	}

	c.writeMu.Lock()
	_, writeErr := c.conn.Write(frame)
	c.writeMu.Unlock()

	if writeErr != nil {
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
		return nil, writeErr
	}

	select {
	case res := <-ch:
		return res.Result, res.Error
	case <-time.After(60 * time.Second):
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
		return nil, fmt.Errorf("timeout waiting for CDP response on method %s", method)
	}
}

// Evaluate evaluates a JavaScript expression within the browser page context.
func (c *CDPClient) Evaluate(expression string) (json.RawMessage, error) {
	params := map[string]interface{}{
		"expression":    expression,
		"returnByValue": true,
		"awaitPromise":  true,
	}
	return c.Send("Runtime.evaluate", params)
}

// Close closes the WebSocket connection.
func (c *CDPClient) Close() error {
	atomic.StoreInt32(&c.closed, 1)
	return c.conn.Close()
}
