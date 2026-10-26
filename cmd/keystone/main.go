package main

import (
	"github.com/viminizer/keystone/internal/config"
	"github.com/viminizer/keystone/internal/engine"
	"github.com/viminizer/keystone/internal/executor"
	"github.com/viminizer/keystone/internal/logging"
	"github.com/viminizer/keystone/internal/server"
	"github.com/viminizer/keystone/internal/workers"
)

func main() {
	settings := config.MustLoad()
	logging.Setup(settings)

	db := engine.MustOpen(settings.AOFPath)
	commandExecutor := executor.New(db)
	backgroundWorkers := workers.New(db)
	backgroundWorkers.Start()

	tcpServer := server.New(settings.Port, commandExecutor)
	tcpServer.MustListenAndServe()
}
