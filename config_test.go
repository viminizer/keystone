package main

import "testing"

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name string
		args []string
		env  map[string]string
		want config
	}{
		{
			name: "defaults",
			want: config{port: 6380, aofPath: "appendonly.aof"},
		},
		{
			name: "environment variables",
			env:  map[string]string{"KEYSTONE_PORT": "7000", "KEYSTONE_AOF_PATH": "/data/keystone.aof"},
			want: config{port: 7000, aofPath: "/data/keystone.aof"},
		},
		{
			name: "flags",
			args: []string{"--port", "8000", "--aof-path", "other.aof"},
			want: config{port: 8000, aofPath: "other.aof"},
		},
		{
			name: "a flag wins over an environment variable",
			args: []string{"--port", "8000"},
			env:  map[string]string{"KEYSTONE_PORT": "7000"},
			want: config{port: 8000, aofPath: "appendonly.aof"},
		},
	}

	for _, test := range tests {
		got, err := loadConfig(test.args, fakeGetenv(test.env))
		if err != nil {
			t.Errorf("%s: got error %v", test.name, err)
			continue
		}
		if got != test.want {
			t.Errorf("%s: got %+v, want %+v", test.name, got, test.want)
		}
	}
}

func TestLoadConfigBadPort(t *testing.T) {
	ports := []string{"abc", "0", "-1", "65536"}

	for _, port := range ports {
		_, err := loadConfig([]string{"--port", port}, fakeGetenv(nil))
		if err == nil {
			t.Errorf("port %q: got no error", port)
		}
		_, err = loadConfig(nil, fakeGetenv(map[string]string{"KEYSTONE_PORT": port}))
		if err == nil {
			t.Errorf("KEYSTONE_PORT=%q: got no error", port)
		}
	}
}

// fakeGetenv works like os.Getenv, but reads from env.
func fakeGetenv(env map[string]string) func(string) string {
	return func(name string) string {
		return env[name]
	}
}
