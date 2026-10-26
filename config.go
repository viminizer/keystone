package main

import (
	"flag"
	"fmt"
	"strconv"
)

// Config holds the settings keystone runs with.
type Config struct {
	port    int
	aofPath string
}

// LoadConfig reads the settings from the command-line flags in args and
// from the environment variables. A flag wins over an environment variable,
// and an environment variable wins over the default.
// getenv reads one environment variable. It is os.Getenv, or a fake one in
// tests.
func LoadConfig(args []string, getenv func(string) string) (*Config, error) {
	flags := flag.NewFlagSet("keystone", flag.ExitOnError)
	// The environment variable becomes the flag's default, so a flag given
	// on the command line still wins over it.
	port := flags.String("port", envOrDefault(getenv, "KEYSTONE_PORT", "6380"),
		"port to listen on, also KEYSTONE_PORT")
	aofPath := flags.String("aof-path", envOrDefault(getenv, "KEYSTONE_AOF_PATH", "appendonly.aof"),
		"path of the append-only file, also KEYSTONE_AOF_PATH")
	flags.Parse(args)

	portNumber, err := strconv.Atoi(*port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return nil, fmt.Errorf("invalid port %q, it must be a number from 1 to 65535", *port)
	}
	return &Config{port: portNumber, aofPath: *aofPath}, nil
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
