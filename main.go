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

		fmt.Println("client connected")

		buffer := make([]byte, 1024)
		for {
			count, err := connection.Read(buffer)
			if err == io.EOF {
				break
			}
			if err != nil {
				fmt.Println("could not read from the client:", err)
				break
			}

			fmt.Printf("received %q\n", buffer[:count])
		}

		connection.Close()
	}
}
