package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/viminizer/keystone/aof"
	"github.com/viminizer/keystone/resp"
	"github.com/viminizer/keystone/store"
)

func main() {
	fmt.Println("keystone is starting")

	settings, err := loadConfig(os.Args[1:], os.Getenv)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Printf("port %d, append-only file %s\n", settings.port, settings.aofPath)

	db := store.New()

	// Load the saved changes before any client can connect, so no client
	// sees the data half loaded. They are not written to the log again.
	loaded, err := aof.Load(settings.aofPath, func(command []string) {
		replayCommand(db, command)
	})
	if err != nil {
		fmt.Println("could not load the append-only file:", err)
		os.Exit(1)
	}
	fmt.Printf("loaded %d commands from %s\n", loaded, settings.aofPath)

	aofLog, err := aof.Open(settings.aofPath)
	if err != nil {
		fmt.Println("could not open the append-only file:", err)
		os.Exit(1)
	}
	engine := NewEngine(db, aofLog)
	executor := NewExecutor(engine)
	go stopOnKillSignal(engine)
	go aofLog.RunSync()
	go runAutoRewrite(engine, aofLog)
	go db.RunActiveExpiry()

	listener, err := net.Listen("tcp", ":"+strconv.Itoa(settings.port))
	if err != nil {
		fmt.Printf("could not listen on port %d: %s\n", settings.port, err)
		os.Exit(1)
	}

	for {
		connection, err := listener.Accept()
		if err != nil {
			fmt.Println("could not accept a connection:", err)
			continue
		}

		go handleConnection(connection, executor)
	}
}

func handleConnection(connection net.Conn, executor *Executor) {
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

		reply := executor.Execute(command)
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

// stopOnKillSignal waits for Ctrl+C (SIGINT) or kill (SIGTERM), and then stops
// keystone cleanly. It never returns, so start it with go.
func stopOnKillSignal(engine *Engine) {
	killSignals := make(chan os.Signal, 1)
	signal.Notify(killSignals, syscall.SIGINT, syscall.SIGTERM)
	received := <-killSignals
	fmt.Printf("received %s, stopping\n", received)

	err := engine.Shutdown()
	if err != nil {
		fmt.Println("could not sync the append-only file:", err)
		os.Exit(1)
	}
	fmt.Println("keystone stopped")
	os.Exit(0)
}
