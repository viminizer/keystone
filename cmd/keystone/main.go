package main

import (
	"fmt"
	"os"

	"github.com/viminizer/keystone/internal/config"
	"github.com/viminizer/keystone/internal/engine"
	"github.com/viminizer/keystone/internal/executor"
	"github.com/viminizer/keystone/internal/server"
	"github.com/viminizer/keystone/internal/workers"
)

func main() {
	fmt.Println("keystone is starting")

	settings := config.MustLoad()
	db := engine.MustOpen(settings.AOFPath)
	commandExecutor := executor.New(db)
	backgroundWorkers := workers.New(db)
	backgroundWorkers.Start()

	tcpServer := server.New(settings.Port, commandExecutor)
	// ListenAndServe only returns if the port could not be opened.
	err := tcpServer.ListenAndServe()
	fmt.Println(err)
	os.Exit(1)
}
