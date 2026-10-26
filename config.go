package main

import (
	"flag"
	"fmt"
	"strconv"
)

// config holds the settings keystone runs with.
type config struct {
	port    int
	aofPath string
}

// loadConfig reads the settings from the command-line flags in args and
// from the environment variables. A flag wins over an environment variable,
// and an environment variable wins over the default.
// getenv reads one environment variable. It is os.Getenv, or a fake one in
// tests.
func loadConfig(args []string, getenv func(string) string) (config, error) {
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
		return config{}, fmt.Errorf("invalid port %q, it must be a number from 1 to 65535", *port)
	}
	return config{port: portNumber, aofPath: *aofPath}, nil
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
