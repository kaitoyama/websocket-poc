package handler

import (
	"github.com/kaitoyama/kaitoyama-server-template/internal/domain"
	"github.com/kaitoyama/kaitoyama-server-template/internal/usecase"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	healthUsecase    usecase.HealthUsecase
	websocketHandler *WebSocketHandler
}

func (h *Handler) GetHealth(c echo.Context) error {
	return h.HealthCheck(c)
}

func NewHandler(dbChecker domain.DatabaseHealthChecker) *Handler {
	return &Handler{
		healthUsecase:    *usecase.NewHealthUsecase(dbChecker),
		websocketHandler: NewWebSocketHandler(),
	}
}

// WebSocket関連のメソッド
func (h *Handler) HandleWebSocket(c echo.Context) error {
	return h.websocketHandler.HandleWebSocket(c)
}

func (h *Handler) GetWebSocketRooms(c echo.Context) error {
	return h.websocketHandler.GetRooms(c)
}
