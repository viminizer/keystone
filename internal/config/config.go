// Package config reads the settings from command-line flags and
// environment variables.
package config

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

// Config holds the settings keystone runs with.
type Config struct {
	Port    int
	AOFPath string
	// LogLevel is the lowest level that is logged.
	LogLevel slog.Level
	// LogFormat is "text" or "json".
	LogFormat string
	// MetricsEnabled says if the metrics are served over HTTP.
	MetricsEnabled bool
	// MetricsPort is the port the metrics are served on.
	MetricsPort int
}

// MustLoad reads the settings from the program's flags and environment
// variables. If they are wrong, it logs the error and exits.
func MustLoad() *Config {
	config, err := Load(os.Args[1:], os.Getenv)
	if err != nil {
		slog.Error("could not load the settings", "error", err)
		os.Exit(1)
	}
	return config
}

// Load reads the settings from the command-line flags in args and
// from the environment variables. A flag wins over an environment variable,
// and an environment variable wins over the default.
// getenv reads one environment variable. It is os.Getenv, or a fake one in
// tests.
func Load(args []string, getenv func(string) string) (*Config, error) {
	flags := flag.NewFlagSet("keystone", flag.ExitOnError)
	// The environment variable becomes the flag's default, so a flag given
	// on the command line still wins over it.
	port := flags.String("port", envOrDefault(getenv, "KEYSTONE_PORT", "6380"),
		"port to listen on, also KEYSTONE_PORT")
	aofPath := flags.String("aof-path", envOrDefault(getenv, "KEYSTONE_AOF_PATH", "appendonly.aof"),
		"path of the append-only file, also KEYSTONE_AOF_PATH")
	logLevel := flags.String("log-level", envOrDefault(getenv, "KEYSTONE_LOG_LEVEL", "info"),
		"lowest level to log: debug, info, warn or error, also KEYSTONE_LOG_LEVEL")
	logFormat := flags.String("log-format", envOrDefault(getenv, "KEYSTONE_LOG_FORMAT", "text"),
		"log format: text or json, also KEYSTONE_LOG_FORMAT")
	metricsEnabled := flags.String("metrics-enabled", envOrDefault(getenv, "KEYSTONE_METRICS_ENABLED", "true"),
		"serve the metrics over HTTP: true or false, also KEYSTONE_METRICS_ENABLED")
	metricsPort := flags.String("metrics-port", envOrDefault(getenv, "KEYSTONE_METRICS_PORT", "9380"),
		"port to serve the metrics on, also KEYSTONE_METRICS_PORT")
	flags.Parse(args)

	portNumber, err := parsePort("port", *port)
	if err != nil {
		return nil, err
	}
	level, err := parseLogLevel(*logLevel)
	if err != nil {
		return nil, err
	}
	format := strings.ToLower(*logFormat)
	if format != "text" && format != "json" {
		return nil, fmt.Errorf("invalid log format %q, it must be text or json", *logFormat)
	}
	enabled, err := strconv.ParseBool(*metricsEnabled)
	if err != nil {
		return nil, fmt.Errorf("invalid metrics enabled %q, it must be true or false", *metricsEnabled)
	}
	metricsPortNumber, err := parsePort("metrics port", *metricsPort)
	if err != nil {
		return nil, err
	}

	config := &Config{
		Port:           portNumber,
		AOFPath:        *aofPath,
		LogLevel:       level,
		LogFormat:      format,
		MetricsEnabled: enabled,
		MetricsPort:    metricsPortNumber,
	}
	return config, nil
}

// parsePort turns text into a port number from 1 to 65535. name says which
// port it is, for the error.
func parsePort(name string, text string) (int, error) {
	port, err := strconv.Atoi(text)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("invalid %s %q, it must be a number from 1 to 65535", name, text)
	}
	return port, nil
}

// parseLogLevel turns "debug", "info", "warn" or "error" into a slog level.
func parseLogLevel(text string) (slog.Level, error) {
	switch strings.ToLower(text) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	}
	return 0, fmt.Errorf("invalid log level %q, it must be debug, info, warn or error", text)
}

// envOrDefault returns the environment variable with the given name, or
// fallback if it is not set.
func envOrDefault(getenv func(string) string, name string, fallback string) string {
	value := getenv(name)
	if value == "" {
		return fallback
	}
	return value
}
