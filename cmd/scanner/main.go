package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"ecdsa-scanner/internal/api"
	"ecdsa-scanner/internal/config"
	"ecdsa-scanner/internal/db"
	"ecdsa-scanner/internal/logger"
	"ecdsa-scanner/internal/notify"
	"ecdsa-scanner/internal/scanner"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	cfg := config.Load()
	appLogger := logger.New(500)

	// Initialize database
	var database db.Database
	var err error

	csvPath := os.Getenv("HITS_CSV_DIR")
	database, err = db.NewCSV(csvPath)
	if err != nil {
		log.Fatalf("CSV storage error: %v", err)
	}
	appLogger.Info("Using CSV hit storage")
	defer database.Close()

	// Initialize notifier
	notifier := notify.New(cfg.PushoverAppToken, cfg.PushoverUserKey)
	if notifier.IsEnabled() {
		appLogger.Info("Pushover notifications enabled")
	}

	// Initialize scanner
	sc, err := scanner.New(database, appLogger, cfg.AlchemyAPIKey, notifier)
	if err != nil {
		log.Fatalf("Scanner error: %v", err)
	}

	// Initialize API
	handler := api.NewHandler(sc, database, appLogger, cfg.AlchemyAPIKey, notifier)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Auto-start scanners
	go func() {
		time.Sleep(2 * time.Second)
		appLogger.Log("Auto-starting scanners...")
		sc.StartAll()
	}()

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		appLogger.Log("Shutting down...")
		sc.StopAll()
		os.Exit(0)
	}()

	// Start HTTP server
	addrs := strings.Split(cfg.BindAddrs, ",")
	for i, addr := range addrs[:len(addrs)-1] {
		listenAddr := fmt.Sprintf("%s:%s", strings.TrimSpace(addr), cfg.Port)
		appLogger.Log("Starting server on %s", listenAddr)
		go func(la string, idx int) {
			if err := http.ListenAndServe(la, mux); err != nil {
				log.Printf("Listener %d error: %v", idx, err)
			}
		}(listenAddr, i)
	}

	lastAddr := fmt.Sprintf("%s:%s", strings.TrimSpace(addrs[len(addrs)-1]), cfg.Port)
	appLogger.Log("Starting server on %s", lastAddr)
	log.Fatal(http.ListenAndServe(lastAddr, mux))
}
