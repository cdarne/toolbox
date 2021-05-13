package main

import (
	"context"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
)

const listenAddr = ":80"
const httpResponse = "HTTP/1.1 200 OK\r\nContent-Length: 3\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nOK\n"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	logger := log.New(os.Stdout, "tcp: ", log.LstdFlags)
	logger.Println("Server is starting...")

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		logger.Fatal(err)
	}
	defer listener.Close()

	logger.Println("Server is ready to handle requests at", listenAddr)
	serve(ctx, logger, listener)
	logger.Println("Server stopped")
}

func serve(ctx context.Context, logger *log.Logger, listener net.Listener) {
	for {
		select {
		case <-ctx.Done():
			logger.Println("Server is shutting down")
			return
		default:
			conn, err := listener.Accept()
			if err != nil {
				logger.Fatal(err)
			}
			logger.Printf("Accepted connection from %s", conn.RemoteAddr().String())

			io.WriteString(conn, httpResponse)
			conn.Close()
		}
	}
}
