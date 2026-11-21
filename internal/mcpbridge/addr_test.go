package mcpbridge

import "testing"

func TestMCPAddr(t *testing.T) {
	env := map[string]string{}
	getenv := func(k string) string { return env[k] }
	for _, tc := range []struct {
		o    Options
		env  map[string]string
		want string
	}{
		{Options{}, nil, "127.0.0.1:7600"},
		{Options{Port: 9000}, nil, "127.0.0.1:9000"},
		{Options{Host: "0.0.0.0"}, nil, "0.0.0.0:7600"},
		{Options{Addr: "localhost:1234", Port: 4321}, nil, "localhost:4321"},
		{Options{Host: "::1", Port: 8000}, nil, "[::1]:8000"},
		{Options{Port: 9000}, map[string]string{"GIODE_MCP_PORT": "9100"}, "127.0.0.1:9100"},
		{Options{}, map[string]string{"GIODE_MCP_ADDR": "10.0.0.2:80", "GIODE_MCP_HOST": "10.0.0.3"}, "10.0.0.3:80"},
		{Options{Addr: "127.0.0.1:0"}, nil, "127.0.0.1:0"},
	} {
		env = tc.env
		got, err := listenAddr(tc.o, getenv)
		if err != nil || got != tc.want {
			t.Errorf("%+v with %v: %q, %v; want %q", tc.o, tc.env, got, err, tc.want)
		}
	}
	env = map[string]string{"GIODE_MCP_PORT": "http"}
	if _, err := listenAddr(Options{}, getenv); err == nil {
		t.Error("a non-numeric port was accepted")
	}
}
