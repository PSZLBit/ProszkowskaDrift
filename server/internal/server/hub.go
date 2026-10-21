package server

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"github.com/PSZLBit/ProszkowskaDrift/server/internal/protocol"
	"github.com/PSZLBit/ProszkowskaDrift/server/internal/session"
)

type outbound struct {
	data       []byte
	closeAfter bool
	code       int
	reason     string
}

type controllerConn struct {
	slot         session.SlotID
	generation   uint64
	ws           *websocket.Conn
	writeTimeout time.Duration

	send      chan outbound
	done      chan struct{}
	closeOnce sync.Once
	closing   atomic.Bool
}

func newControllerConn(slot session.SlotID, generation uint64, ws *websocket.Conn, writeTimeout time.Duration) *controllerConn {
	c := &controllerConn{
		slot:         slot,
		generation:   generation,
		ws:           ws,
		writeTimeout: writeTimeout,
		send:         make(chan outbound, 64),
		done:         make(chan struct{}),
	}
	go c.writer()
	return c
}

func (c *controllerConn) enqueue(data []byte) bool {
	select {
	case c.send <- outbound{data: data}:
		return true
	case <-c.done:
		return false
	default:
		return false
	}
}

func (c *controllerConn) enqueueClose(data []byte, code int, reason string) bool {
	select {
	case c.send <- outbound{data: data, closeAfter: true, code: code, reason: reason}:
		c.closing.Store(true)
		return true
	case <-c.done:
		return false
	default:
		return false
	}
}

func (c *controllerConn) isClosing() bool {
	return c.closing.Load()
}

func (c *controllerConn) writer() {
	for {
		select {
		case out := <-c.send:
			if len(out.data) > 0 {
				_ = c.ws.SetWriteDeadline(time.Now().Add(c.writeTimeout))
				if err := c.ws.WriteMessage(websocket.TextMessage, out.data); err != nil {
					c.finish(websocket.CloseInternalServerErr, "write failed")
					return
				}
			}
			if out.closeAfter {
				c.finish(out.code, out.reason)
				return
			}
		case <-c.done:
			return
		}
	}
}

func (c *controllerConn) closeNow(code int, reason string) {
	c.finish(code, reason)
}

func (c *controllerConn) finish(code int, reason string) {
	c.closing.Store(true)
	c.closeOnce.Do(func() {
		deadline := time.Now().Add(c.writeTimeout)
		_ = c.ws.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(code, reason),
			deadline,
		)
		_ = c.ws.Close()
		close(c.done)
	})
}

type hub struct {
	mu    sync.RWMutex
	conns map[session.SlotID]*controllerConn
}

func newHub() *hub {
	return &hub{conns: make(map[session.SlotID]*controllerConn)}
}

func (h *hub) add(conn *controllerConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.conns[conn.slot] = conn
}

func (h *hub) remove(slot session.SlotID, conn *controllerConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if current, ok := h.conns[slot]; ok && current == conn {
		delete(h.conns, slot)
	}
}

func (h *hub) get(slot session.SlotID) *controllerConn {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.conns[slot]
}

func (h *hub) sendToSlot(slot session.SlotID, data []byte) bool {
	conn := h.get(slot)
	if conn == nil {
		return false
	}
	if !conn.enqueue(data) {
		conn.closeNow(websocket.ClosePolicyViolation, "slow client")
		return false
	}
	return true
}

func (h *hub) broadcast(data []byte) {
	h.mu.RLock()
	conns := make([]*controllerConn, 0, len(h.conns))
	for _, conn := range h.conns {
		conns = append(conns, conn)
	}
	h.mu.RUnlock()
	for _, conn := range conns {
		if !conn.enqueue(data) {
			conn.closeNow(websocket.ClosePolicyViolation, "slow client")
		}
	}
}

func (h *hub) closeSlotIfGeneration(slot session.SlotID, generation uint64, code int, reason string) {
	if conn := h.get(slot); conn != nil && conn.generation == generation {
		conn.closeNow(code, reason)
	}
}

func (h *hub) closeSlotWithError(slot session.SlotID, code, message string, closeCode int, closeReason string) {
	conn := h.get(slot)
	if conn == nil {
		return
	}
	if !conn.enqueueClose(errorFrame(code, message), closeCode, closeReason) {
		conn.closeNow(closeCode, closeReason)
	}
}

func (h *hub) closeAll(code int, reason string) {
	h.mu.RLock()
	conns := make([]*controllerConn, 0, len(h.conns))
	for _, conn := range h.conns {
		conns = append(conns, conn)
	}
	h.mu.RUnlock()
	for _, conn := range conns {
		conn.closeNow(code, reason)
	}
}

// closeAllGraceful drains already queued messages before closing. It is used
// when a final phase or result frame must reach the controllers.
func (h *hub) closeAllGraceful(code int, reason string) {
	h.mu.RLock()
	conns := make([]*controllerConn, 0, len(h.conns))
	for _, conn := range h.conns {
		conns = append(conns, conn)
	}
	h.mu.RUnlock()
	for _, conn := range conns {
		if !conn.enqueueClose(nil, code, reason) {
			conn.closeNow(code, reason)
		}
	}
}

func (h *hub) count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.conns)
}

func errorFrame(code, message string) []byte {
	payload, err := protocol.EncodeMessage(protocol.Version, protocol.TypeError, nil, protocol.ErrorData{
		Code:    code,
		Message: message,
	})
	if err != nil {
		return nil
	}
	return payload
}
