package config

import (
	"log/slog"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name string
		args []string
		env  map[string]string
		want *Config
	}{
		{
			name: "defaults",
			want: &Config{
				Port: 6380, AOFPath: "appendonly.aof", LogLevel: slog.LevelInfo, LogFormat: "text",
				HTTPEnabled: true, HTTPPort: 9380,
			},
		},
		{
			name: "environment variables",
			env: map[string]string{
				"KEYSTONE_PORT":         "7000",
				"KEYSTONE_AOF_PATH":     "/data/keystone.aof",
				"KEYSTONE_LOG_LEVEL":    "debug",
				"KEYSTONE_LOG_FORMAT":   "json",
				"KEYSTONE_HTTP_ENABLED": "false",
				"KEYSTONE_HTTP_PORT":    "9000",
			},
			want: &Config{
				Port: 7000, AOFPath: "/data/keystone.aof", LogLevel: slog.LevelDebug, LogFormat: "json",
				HTTPEnabled: false, HTTPPort: 9000,
			},
		},
		{
			name: "flags",
			args: []string{
				"--port", "8000", "--aof-path", "other.aof", "--log-level", "WARN", "--log-format", "JSON",
				"--http-enabled", "0", "--http-port", "9100",
			},
			want: &Config{
				Port: 8000, AOFPath: "other.aof", LogLevel: slog.LevelWarn, LogFormat: "json",
				HTTPEnabled: false, HTTPPort: 9100,
			},
		},
		{
			name: "a flag wins over an environment variable",
			args: []string{"--port", "8000", "--log-level", "error", "--http-enabled=true"},
			env:  map[string]string{"KEYSTONE_PORT": "7000", "KEYSTONE_LOG_LEVEL": "debug", "KEYSTONE_HTTP_ENABLED": "false"},
			want: &Config{
				Port: 8000, AOFPath: "appendonly.aof", LogLevel: slog.LevelError, LogFormat: "text",
				HTTPEnabled: true, HTTPPort: 9380,
			},
		},
	}

	for _, test := range tests {
		got, err := Load(test.args, fakeGetenv(test.env))
		if err != nil {
			t.Errorf("%s: got error %v", test.name, err)
			continue
		}
		if *got != *test.want {
			t.Errorf("%s: got %+v, want %+v", test.name, got, test.want)
		}
	}
}

func TestLoadBadPort(t *testing.T) {
	ports := []string{"abc", "0", "-1", "65536"}

	for _, port := range ports {
		_, err := Load([]string{"--port", port}, fakeGetenv(nil))
		if err == nil {
			t.Errorf("port %q: got no error", port)
		}
		_, err = Load(nil, fakeGetenv(map[string]string{"KEYSTONE_PORT": port}))
		if err == nil {
			t.Errorf("KEYSTONE_PORT=%q: got no error", port)
		}
	}
}

func TestLoadBadSettings(t *testing.T) {
	tests := [][]string{
		{"--log-level", "verbose"},
		{"--log-level", ""},
		{"--log-format", "xml"},
		{"--http-enabled", "maybe"},
		{"--http-port", "0"},
		{"--http-port", "abc"},
	}

	for _, args := range tests {
		_, err := Load(args, fakeGetenv(nil))
		if err == nil {
			t.Errorf("Load(%q): got no error", args)
		}
	}
}

// fakeGetenv works like os.Getenv, but reads from env.
func fakeGetenv(env map[string]string) func(string) string {
	return func(name string) string {
		return env[name]
	}
}
