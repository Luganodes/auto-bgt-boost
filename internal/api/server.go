package api

import (
	"bgt_boost/internal/config"
	"bgt_boost/internal/repository"
	"fmt"
	"net/http"
	"time"

	_ "github.com/joho/godotenv/autoload"
)

type Server struct {
	dbRepository *repository.DbRepository
	config       *config.Config
}

func NewServer(config *config.Config, dbRepository *repository.DbRepository) *http.Server {
	NewServer := &Server{
		dbRepository: dbRepository,
		config:       config,
	}

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.API_PORT),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}
