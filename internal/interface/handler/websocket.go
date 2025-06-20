package handler

import (
	"net/http"

	"github.com/kaitoyama/kaitoyama-server-template/internal/usecase"
	"github.com/labstack/echo/v4"
)

// WebSocketHandler handles WebSocket-related HTTP requests
type WebSocketHandler struct {
	wsUsecase *usecase.WebSocketUsecase
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler() *WebSocketHandler {
	return &WebSocketHandler{
		wsUsecase: usecase.NewWebSocketUsecase(),
	}
}

// HandleWebSocket handles WebSocket upgrade requests
func (h *WebSocketHandler) HandleWebSocket(c echo.Context) error {
	return h.wsUsecase.HandleWebSocketConnection(c)
}

// GetRooms returns information about active WebSocket rooms
func (h *WebSocketHandler) GetRooms(c echo.Context) error {
	info := h.wsUsecase.GetRoomInfo()
	return c.JSON(http.StatusOK, map[string]interface{}{
		"rooms": info,
	})
}
