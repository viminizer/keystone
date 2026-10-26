package main

import (
	"log/slog"
	"os"

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
	slog.Info("keystone is starting",
		"port", settings.Port,
		"aof_path", settings.AOFPath,
		"log_level", settings.LogLevel.String(),
		"log_format", settings.LogFormat)

	db := engine.MustOpen(settings.AOFPath)
	commandExecutor := executor.New(db)
	backgroundWorkers := workers.New(db)
	backgroundWorkers.Start()

	tcpServer := server.New(settings.Port, commandExecutor)
	// ListenAndServe only returns if the port could not be opened.
	err := tcpServer.ListenAndServe()
	slog.Error("could not start the server", "error", err)
	os.Exit(1)
}
