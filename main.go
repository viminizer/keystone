package main

import (
	"fmt"
	"os"
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

	server := NewServer(config.port, executor)
	// ListenAndServe only returns if the port could not be opened.
	err = server.ListenAndServe()
	fmt.Println(err)
	os.Exit(1)
}
