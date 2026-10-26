package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("keystone is starting")

	config := MustLoadConfig()
	engine := MustOpenEngine(config.aofPath)
	executor := NewExecutor(engine)
	workers := NewWorkers(engine)
	workers.Start()

	server := NewServer(config.port, executor)
	// ListenAndServe only returns if the port could not be opened.
	err := server.ListenAndServe()
	fmt.Println(err)
	os.Exit(1)
}
