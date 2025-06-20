package main

import (
	"database/sql"
	"embed"
	"io/fs"
	"log"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/kaitoyama/kaitoyama-server-template/internal/infrastructure/db"
	"github.com/kaitoyama/kaitoyama-server-template/internal/interface/handler"
	"github.com/kaitoyama/kaitoyama-server-template/openapi"
	"github.com/labstack/echo/v4"
)

//go:embed openapi/swagger.yml
var swaggerFS embed.FS

//go:embed static/*
var staticFS embed.FS

func SetupRouter(database *sql.DB) *echo.Echo {
	// Initialize Echo
	e := echo.New()

	// Load OpenAPI spec from embedded file
	swaggerBytes, err := swaggerFS.ReadFile("openapi/swagger.yml")
	if err != nil {
		log.Fatalf("Failed to read embedded swagger file: %v", err)
	}
	loader := openapi3.NewLoader()
	swagger, err := loader.LoadFromData(swaggerBytes)
	if err != nil {
		log.Fatalf("Failed to load swagger spec: %v", err)
	}
	if err = swagger.Validate(loader.Context); err != nil {
		log.Fatalf("Swagger spec validation error: %v", err)
	}

	// Apply middleware to validate incoming requests against the OpenAPI spec

	dbChecker := db.NewDBHealthChecker(database)
	healthHandler := handler.NewHandler(dbChecker)

	// oapi-codegenのRegisterHandlersを使用
	openapi.RegisterHandlers(e, healthHandler)

	// WebSocket endpoints (not part of OpenAPI spec)
	e.GET("/ws", healthHandler.HandleWebSocket)
	e.GET("/ws/rooms", healthHandler.GetWebSocketRooms)

	// Static files from embedded filesystem
	staticSubFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatalf("Failed to create sub filesystem for static files: %v", err)
	}
	e.GET("/static/*", echo.WrapHandler(http.FileServer(http.FS(staticSubFS))))

	return e
}
