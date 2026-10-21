package server

import (
	"errors"
	"time"

	"github.com/gorilla/websocket"

	"github.com/PSZLBit/ProszkowskaDrift/server/internal/protocol"
)

func writeMessage(ws *websocket.Conn, messageType string, data any, timeout time.Duration) error {
	payload, err := protocol.EncodeMessage(protocol.Version, messageType, nil, data)
	if err != nil {
		return err
	}
	return writePayload(ws, payload, timeout)
}

func writePayload(ws *websocket.Conn, payload []byte, timeout time.Duration) error {
	_ = ws.SetWriteDeadline(time.Now().Add(timeout))
	return ws.WriteMessage(websocket.TextMessage, payload)
}

func readEnvelope(ws *websocket.Conn, maxBytes int64, timeout time.Duration) (protocol.Envelope, error) {
	ws.SetReadLimit(maxBytes)
	_ = ws.SetReadDeadline(time.Now().Add(timeout))
	messageType, raw, err := ws.ReadMessage()
	if err != nil {
		return protocol.Envelope{}, err
	}
	if messageType != websocket.TextMessage {
		return protocol.Envelope{}, protocol.NewError(protocol.CodeInvalidInput, "Protocol messages must be JSON text frames.")
	}
	return protocol.ParseEnvelope(raw)
}

func protocolError(err error) *protocol.Error {
	var pe *protocol.Error
	if errors.As(err, &pe) {
		return pe
	}
	return protocol.NewError(protocol.CodeInternalError, "Internal server error.")
}
