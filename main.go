package main

import (
	"fmt"
	"io"
	"net"
	"os"
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

	buffer := make([]byte, 1024)
	for {
		count, err := connection.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("could not read from %s: %s\n", address, err)
			break
		}

		fmt.Printf("received %q from %s\n", buffer[:count], address)
	}

	connection.Close()
}
