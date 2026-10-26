package main

import (
	"github.com/viminizer/keystone/internal/config"
	"github.com/viminizer/keystone/internal/engine"
	"github.com/viminizer/keystone/internal/executor"
	"github.com/viminizer/keystone/internal/httpserver"
	"github.com/viminizer/keystone/internal/logging"
	"github.com/viminizer/keystone/internal/metrics"
	"github.com/viminizer/keystone/internal/server"
	"github.com/viminizer/keystone/internal/workers"
)

func main() {
	settings := config.MustLoad()
	logging.Setup(settings)

	db := engine.MustOpen(settings.AOFPath)
	metricsRegistry := metrics.NewRegistry()
	commandExecutor := executor.New(db, metricsRegistry.CommandStats)
	backgroundWorkers := workers.New(db)
	backgroundWorkers.Start()

	httpserver.New(settings).
		Get("metrics", metricsRegistry.HandleMetrics).
		MustStart()

	tcpServer := server.New(settings.Port, commandExecutor)
	tcpServer.MustListenAndServe()
}
