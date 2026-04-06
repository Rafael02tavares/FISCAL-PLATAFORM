package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Rafael02tavares/FISCAL-PLATAFORM/backend/internal/config"
	"github.com/Rafael02tavares/FISCAL-PLATAFORM/backend/internal/database"
	"github.com/Rafael02tavares/FISCAL-PLATAFORM/backend/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if cfg.Port == "" {
		log.Fatal("load config: PORT is required")
	}

	db, err := database.NewPostgres(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer db.Close()

	log.Printf("database connected successfully")
	log.Printf("starting server in %s environment on port %s", cfg.AppEnv, cfg.Port)

	handler := server.New(cfg, db)

	httpServer := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErr := make(chan error, 1)

	go func() {
		err := httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	select {
	case err := <-serverErr:
		if err != nil {
			log.Fatalf("listen and serve: %v", err)
		}
	case <-shutdownSignal():
		log.Println("shutdown signal received")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
		} else {
			log.Println("server stopped")
		}
	}
}

func shutdownSignal() <-chan struct{} {
	stopCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	done := make(chan struct{})

	go func() {
		defer stop()
		<-stopCtx.Done()
		close(done)
	}()

	return done
}