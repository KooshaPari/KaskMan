package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kooshapari/kaskmanager-rd-platform/internal/chat"
	"github.com/sirupsen/logrus"
)

var (
	port     = flag.Int("port", 8080, "Server port")
	logLevel = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	dev      = flag.Bool("dev", false, "Development mode")
)

func main() {
	flag.Parse()

	// Setup logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	
	level, err := logrus.ParseLevel(*logLevel)
	if err != nil {
		logger.Fatal("Invalid log level:", *logLevel)
	}
	logger.SetLevel(level)

	if *dev {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
			ForceColors:   true,
		})
		logger.SetLevel(logrus.DebugLevel)
	}

	logger.WithFields(logrus.Fields{
		"port":      *port,
		"log_level": *logLevel,
		"dev_mode":  *dev,
	}).Info("Starting KaskMan Chat Server")

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.WithField("signal", sig).Info("Received shutdown signal")
		cancel()
	}()

	// Initialize components  
	chatServer := chat.NewChatServer(logger, nil, *port)

	// Start server
	logger.WithField("port", *port).Info("Chat server starting")
	fmt.Printf("\n🧠 KaskMan Chat Server Starting\n")
	fmt.Printf("📡 Server: http://localhost:%d\n", *port)
	fmt.Printf("💬 Chat Interface: http://localhost:%d/\n", *port)
	fmt.Printf("🔗 WebSocket: ws://localhost:%d/ws\n", *port)
	fmt.Printf("📊 Health Check: http://localhost:%d/api/v1/health\n", *port)
	fmt.Printf("📈 Metrics: http://localhost:%d/api/v1/metrics\n\n", *port)

	if err := chatServer.StartServer(ctx); err != nil {
		logger.WithError(err).Fatal("Server failed to start")
	}

	logger.Info("KaskMan Chat Server stopped")
}