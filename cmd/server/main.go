package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"ponches/internal/api"
	"ponches/internal/config"
	"ponches/internal/hikvision"
	"ponches/internal/realtime"
	"ponches/internal/setup"
	"ponches/internal/store"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Setup Logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Load Config
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load config")
	}

	// Init Store
	repo, err := store.NewSQLiteStore(cfg.DBPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize store")
	}

	// Overlay persisted configuration from database before wiring runtime services.
	if values, err := repo.GetAllConfig(context.Background()); err == nil {
		config.ApplyOverrides(cfg, values)
	} else {
		log.Warn().Err(err).Msg("Failed to load persisted configuration")
	}

	// Initialize default admin user
	if err := setup.InitDefaultAdmin(repo); err != nil {
		log.Warn().Err(err).Msg("Failed to initialize default admin user")
	}

	// Init Hub
	hub := realtime.NewHub()
	go hub.Run()

	// Init Hikvision Event Listener (if configured)
	if cfg.HikvisionIP != "" && cfg.HikvisionUsername != "" {
		listener := hikvision.NewEventListener(cfg.HikvisionIP, 80, cfg.HikvisionUsername, cfg.HikvisionPassword)

		// Add handler to broadcast events via WebSocket and save to DB
		listener.AddHandler(func(event *hikvision.AttendanceEvent) {
			// Save to database
			storeEvent := &store.AttendanceEvent{
				DeviceID:   event.DeviceID,
				EmployeeNo: event.EmployeeNo,
				Timestamp:  event.Timestamp,
				Type:       event.EventType,
			}
			repo.SaveEvent(context.Background(), storeEvent)

			// Get employee name for broadcast
			emp, err := repo.GetEmployeeByNo(context.Background(), event.EmployeeNo)
			employeeName := event.EmployeeNo
			if err == nil && emp != nil {
				employeeName = emp.FirstName + " " + emp.LastName
			}

			// Broadcast to WebSocket clients
			hub.BroadcastAttendanceEvent(event.EmployeeNo, employeeName, event.DeviceID, event.Timestamp)

			log.Info().Str("employeeNo", event.EmployeeNo).Time("timestamp", event.Timestamp).Msg("Attendance event received")
		})

		// Start listener in background
		go func() {
			if err := listener.Start(); err != nil {
				log.Warn().Err(err).Msg("Event listener stopped")
			}
		}()
		log.Info().Str("device", cfg.HikvisionIP).Msg("Hikvision event listener started")
	}

	// Init Server
	srv := api.NewServer(cfg, repo, hub)

	server := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: srv.Router,
	}

	// Run Server
	go func() {
		log.Info().Msgf("Server starting on port %s", cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed")
		}
	}()

	// Wait for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Info().Msg("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server stopped gracefully")
}
