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
			want: &Config{Port: 6380, AOFPath: "appendonly.aof", LogLevel: slog.LevelInfo, LogFormat: "text"},
		},
		{
			name: "environment variables",
			env: map[string]string{
				"KEYSTONE_PORT":       "7000",
				"KEYSTONE_AOF_PATH":   "/data/keystone.aof",
				"KEYSTONE_LOG_LEVEL":  "debug",
				"KEYSTONE_LOG_FORMAT": "json",
			},
			want: &Config{Port: 7000, AOFPath: "/data/keystone.aof", LogLevel: slog.LevelDebug, LogFormat: "json"},
		},
		{
			name: "flags",
			args: []string{"--port", "8000", "--aof-path", "other.aof", "--log-level", "WARN", "--log-format", "JSON"},
			want: &Config{Port: 8000, AOFPath: "other.aof", LogLevel: slog.LevelWarn, LogFormat: "json"},
		},
		{
			name: "a flag wins over an environment variable",
			args: []string{"--port", "8000", "--log-level", "error"},
			env:  map[string]string{"KEYSTONE_PORT": "7000", "KEYSTONE_LOG_LEVEL": "debug"},
			want: &Config{Port: 8000, AOFPath: "appendonly.aof", LogLevel: slog.LevelError, LogFormat: "text"},
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

func TestLoadBadLogSettings(t *testing.T) {
	tests := [][]string{
		{"--log-level", "verbose"},
		{"--log-level", ""},
		{"--log-format", "xml"},
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
