package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

func setupSignals(socketPath string) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-interrupt
		_ = os.RemoveAll(filepath.Dir(socketPath))
		os.Exit(0)
	}()
}

func setupSocket(socketPath string) (net.Listener, error) {
	_ = os.RemoveAll(filepath.Dir(socketPath))
	if err := os.MkdirAll(filepath.Dir(socketPath), 0700); err != nil {
		return nil, fmt.Errorf("failed to create directory %q: %v", filepath.Dir(socketPath), err)
	}
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %q: %v", socketPath, err)
	}

	log.Printf("Listening on: unix://%s", socketPath)
	return listener, nil
}

func main() {
	log.Println("Start EPA plugin.")
	const socketPath = "/var/run/scope/plugins/epa/epa.sock"

	// Handle the exit signal
	setupSignals(socketPath)

	// Create socket listener
	listener, err := setupSocket(socketPath)
	if err != nil {
		log.Fatalln("Failed to setup socket: ", err)
	}

	// Create defer to remove socket in the end
	defer func() {
		_ = listener.Close()
		err := os.RemoveAll(filepath.Dir(socketPath))
		if err != nil {
			log.Fatalln("Failed to remove socket")
		}
	}()

	// Initialize a plugin implementation
	plugin := &Plugin{}

	// Start server
	http.HandleFunc("/report", plugin.Report)
	if err := http.Serve(listener, nil); err != nil {
		log.Fatalln("Failed to start server")
	}
}
