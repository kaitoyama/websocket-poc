package usecase

import (
	"context"
	"fmt"
	"net/http"

	"github.com/coder/websocket"
	"github.com/kaitoyama/kaitoyama-server-template/internal/domain"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

type WebSocketUsecase struct {
	roomManager *domain.RoomManager
}

func NewWebSocketUsecase() *WebSocketUsecase {
	return &WebSocketUsecase{
		roomManager: domain.NewRoomManager(),
	}
}

// HandleWebSocketConnection handles WebSocket connections
func (w *WebSocketUsecase) HandleWebSocketConnection(c echo.Context) error {
	// Get room ID from query parameter
	roomID := c.QueryParam("room_id")
	if roomID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "room_id is required")
	}

	// Upgrade connection to WebSocket
	conn, err := websocket.Accept(c.Response().Writer, c.Request(), &websocket.AcceptOptions{
		OriginPatterns: []string{"*"}, // In production, specify allowed origins
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to upgrade WebSocket connection")
		return fmt.Errorf("failed to upgrade connection: %w", err)
	}
	defer conn.Close(websocket.StatusInternalError, "connection closed")

	// Create connection wrapper
	wsConn := &domain.WebSocketConnection{
		Conn:   conn,
		RoomID: roomID,
		UserID: c.QueryParam("user_id"), // Optional user identifier
	}

	// Get or create room
	room := w.roomManager.GetOrCreateRoom(roomID)

	// Generate connection ID
	connID := fmt.Sprintf("%s_%d", roomID, len(room.Connections))

	// Add connection to room
	room.AddConnection(connID, wsConn)
	defer func() {
		room.RemoveConnection(connID)
		// Clean up empty rooms
		w.roomManager.RemoveRoom(roomID)
	}()

	log.Info().
		Str("room_id", roomID).
		Str("connection_id", connID).
		Msg("WebSocket connection established")

	// Keep connection alive and handle incoming messages
	ctx := context.Background()
	for {
		// Read message from client (for potential future use)
		_, message, err := conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
				websocket.CloseStatus(err) == websocket.StatusGoingAway {
				log.Info().
					Str("room_id", roomID).
					Str("connection_id", connID).
					Msg("WebSocket connection closed normally")
			} else {
				log.Error().
					Err(err).
					Str("room_id", roomID).
					Str("connection_id", connID).
					Msg("WebSocket connection error")
			}
			break
		}

		log.Debug().
			Str("room_id", roomID).
			Str("connection_id", connID).
			Str("message", string(message)).
			Msg("Received WebSocket message")
	}

	return nil
}

// GetRoomInfo returns information about active rooms (for debugging/monitoring)
func (w *WebSocketUsecase) GetRoomInfo() map[string]interface{} {
	w.roomManager.Mutex.RLock()
	defer w.roomManager.Mutex.RUnlock()

	info := make(map[string]interface{})
	for roomID, room := range w.roomManager.Rooms {
		room.Mutex.RLock()
		info[roomID] = map[string]interface{}{
			"connections": len(room.Connections),
			"counter":     room.Counter,
			"started":     room.Started,
		}
		room.Mutex.RUnlock()
	}

	return info
}
