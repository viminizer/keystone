package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/viminizer/keystone/resp"
)

func main() {
	fmt.Println("keystone is starting")

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

		go handleConnection(connection)
	}
}

func handleConnection(connection net.Conn) {
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

		reply := runCommand(command)
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
