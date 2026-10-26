package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	fmt.Println("keystone is starting")

	config, err := LoadConfig(os.Args[1:], os.Getenv)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Printf("port %d, append-only file %s\n", config.port, config.aofPath)

	engine, err := OpenEngine(config.aofPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	executor := NewExecutor(engine)
	workers := NewWorkers(engine)
	workers.Start()
	go stopOnKillSignal(engine)

	server := NewServer(config.port, executor)
	// ListenAndServe only returns if the port could not be opened.
	err = server.ListenAndServe()
	fmt.Println(err)
	os.Exit(1)
}

// stopOnKillSignal waits for Ctrl+C (SIGINT) or kill (SIGTERM), and then stops
// keystone cleanly. It never returns, so start it with go.
func stopOnKillSignal(engine *Engine) {
	killSignals := make(chan os.Signal, 1)
	signal.Notify(killSignals, syscall.SIGINT, syscall.SIGTERM)
	received := <-killSignals
	fmt.Printf("received %s, stopping\n", received)

	err := engine.Shutdown()
	if err != nil {
		fmt.Println("could not sync the append-only file:", err)
		os.Exit(1)
	}
	fmt.Println("keystone stopped")
	os.Exit(0)
}
