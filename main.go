package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/viminizer/keystone/aof"
	"github.com/viminizer/keystone/resp"
	"github.com/viminizer/keystone/store"
)

const aofPath = "appendonly.aof"

func main() {
	fmt.Println("keystone is starting")

	db := store.New()

	// Load the saved changes before any client can connect, so no client
	// sees the data half loaded. They are not written to the log again.
	loaded, err := aof.Load(aofPath, func(command []string) {
		runCommand(command, db, nil)
	})
	if err != nil {
		fmt.Println("could not load the append-only file:", err)
		os.Exit(1)
	}
	fmt.Printf("loaded %d commands from %s\n", loaded, aofPath)

	aofLog, err := aof.Open(aofPath)
	if err != nil {
		fmt.Println("could not open the append-only file:", err)
		os.Exit(1)
	}
	go aofLog.RunSync()
	go db.RunActiveExpiry()

	listener, err := net.Listen("tcp", ":6380")
	if err != nil {
		fmt.Println("could not listen on port 6380:", err)
		os.Exit(1)
	}

	for {
		connection, err := listener.Accept()
		if err != nil {
			fmt.Println("could not accept a connection:", err)
			continue
		}

		go handleConnection(connection, db, aofLog)
	}
}

func handleConnection(connection net.Conn, db *store.Store, aofLog *aof.Log) {
	address := connection.RemoteAddr().String()
	fmt.Println("client connected from", address)

	reader := bufio.NewReader(connection)
	for {
		command, err := resp.ReadCommand(reader)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("could not read from %s: %s\n", address, err)
			break
		}

		fmt.Printf("received %q from %s\n", command, address)

		reply := runCommand(command, db, aofLog)
		if reply == "" {
			continue
		}
		_, err = connection.Write([]byte(reply))
		if err != nil {
			fmt.Printf("could not write to %s: %s\n", address, err)
			break
		}
	}

	connection.Close()
}
