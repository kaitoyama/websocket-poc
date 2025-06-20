package main

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/kaitoyama/kaitoyama-server-template/internal/infrastructure/config"
)

func initDB(cfg *config.Config) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
	)

	var database *sql.DB
	var err error
	
	// Retry connection up to 30 times with 1-second interval
	for i := 0; i < 30; i++ {
		database, err = sql.Open("mysql", dsn)
		if err != nil {
			log.Warn().Err(err).Msgf("Failed to open database connection, attempt %d/30", i+1)
			time.Sleep(1 * time.Second)
			continue
		}

		if err = database.Ping(); err != nil {
			log.Warn().Err(err).Msgf("Failed to ping database, attempt %d/30", i+1)
			database.Close()
			time.Sleep(1 * time.Second)
			continue
		}

		log.Info().Msg("Successfully connected to database")
		return database, nil
	}

	return nil, fmt.Errorf("failed to connect to database after 30 attempts: %w", err)
}

func main() {
	// Configure zerolog
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	// Load configuration
	cfg := config.LoadConfig()

	// Initialize database
	database, err := initDB(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer database.Close()

	// Create Echo instance
	e := SetupRouter(database)

	// start server
	e.Logger.Fatal(e.Start(":" + cfg.Port))
}
