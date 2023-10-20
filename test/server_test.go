package test

import (
	"bytes"
	"net"
	"testing"

	"github.com/countersoda/godis/app"
)

var tcp app.Server

func init() {
	// Start the new server
	tcp = app.NewServer(":6379")
	go func() {
		tcp.Run()
	}()
}

func TestServerRunning(t *testing.T) {
	// Simply check that the server is up and can
	// accept connections.
	server := struct {
		protocol string
		addr     string
	}{"tcp", ":6379"}
	conn, err := net.Dial(server.protocol, server.addr)
	if err != nil {
		t.Error("could not connect to server: ", err)
	}
	defer conn.Close()
}

func testServerRequest(t *testing.T) {
	servers := struct {
		protocol string
		addr     string
	}{"tcp", ":6379"}

	tt := []struct {
		test    string
		payload []byte
		want    []byte
	}{
		{"Sending a simple request returns result", []byte("hello world\n"), []byte("Request received: hello world")},
		{"Sending another simple request works", []byte("goodbye world\n"), []byte("Request received: goodbye world")},
	}

	for _, tc := range tt {
		t.Run(tc.test, func(t *testing.T) {
			conn, err := net.Dial(servers.protocol, servers.addr)
			if err != nil {
				t.Error("could not connect to server: ", err)
			}
			defer conn.Close()

			if _, err := conn.Write(tc.payload); err != nil {
				t.Error("could not write payload to server:", err)
			}

			out := make([]byte, 1024)
			if _, err := conn.Read(out); err == nil {
				if bytes.Compare(out, tc.want) == 0 {
					t.Error("response did match expected output")
				}
			} else {
				t.Error("could not read from connection")
			}
		})
	}
}
