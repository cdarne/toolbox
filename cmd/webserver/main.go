package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"
)

/*
curl -vv http://localhost:1984 -X POST http://localhost:1984
ab -n 100 -c 10 -k -p ~/sample.json -T "application/json; charset=utf-8" http://localhost:1984/

HTTPS
curl --http1.1 --cacert certs/ca.pem -vv https://127.0.0.1:1984 -X POST https://127.0.0.1:1984
ab -n 100 -c 10 -k -p ~/sample.json -T "application/json; charset=utf-8" https://127.0.0.1:1984/
*/
type key int

const (
	requestIDKey key = 0
	listenAddr       = "127.0.0.1:1984"
)

var caCert = flag.String("ca-cert", "", "CA certificate path.")
var serverCert = flag.String("server-cert", "", "Server certificate path.")
var serverKey = flag.String("server-key", "", "Server key path.")

func index() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "OK")
	})
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	flag.Parse()

	logger := log.New(os.Stdout, "http: ", log.LstdFlags)
	logger.Println("Server is starting...")
	server := setupServer(logger)

	startServer(logger, server)
	logger.Println("Server is ready to handle requests at", listenAddr)

	<-ctx.Done()
	// stop handling the Interrupt signal. This restores the default go behaviour (exit) in case of a second Interrupt
	stop()

	logger.Println("Server is shutting down")
	if err := shutdownServer(logger, server); err != nil {
		logger.Fatalf("Could not gracefully shutdown the server: %v\n", err)
	}
	logger.Println("Server stopped")
}

func setupServer(logger *log.Logger) *http.Server {
	router := http.NewServeMux()
	router.Handle("/", index())

	nextRequestID := func() string {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}

	tcpLogger := log.New(os.Stdout, "tcp: ", log.LstdFlags)

	server := &http.Server{
		Addr:         listenAddr,
		Handler:      tracing(nextRequestID)(logging(logger)(router)),
		ErrorLog:     logger,
		ConnState:    connLogging(tcpLogger),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	if useSSL() {
		tlsConfig, err := setupTLS(*serverCert, *serverKey, *caCert)
		if err != nil {
			logger.Fatalln(err)
		}
		server.TLSConfig = tlsConfig
	}

	return server
}

func startServer(logger *log.Logger, server *http.Server) {
	go func() {
		var err error
		if useSSL() {
			err = server.ListenAndServeTLS("", "")
		} else {
			err = server.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			logger.Fatal(err)
		}
	}()
}

func shutdownServer(logger *log.Logger, server *http.Server) error {
	ctxShutDown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	server.SetKeepAlivesEnabled(false)
	return server.Shutdown(ctxShutDown)
}

func logging(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				requestID, ok := r.Context().Value(requestIDKey).(string)
				if !ok {
					requestID = "unknown"
				}
				logger.Println(requestID, r.Proto, r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent(), r.Header)
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func tracing(nextRequestID func() string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-Id")
			if requestID == "" {
				requestID = nextRequestID()
			}
			ctx := context.WithValue(r.Context(), requestIDKey, requestID)
			w.Header().Set("X-Request-Id", requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func connLogging(logger *log.Logger) func(net.Conn, http.ConnState) {
	return func(conn net.Conn, connState http.ConnState) {
		logger.Printf("conn %s [%s]\n", conn.RemoteAddr(), connState.String())
	}
}

func useSSL() bool {
	return *caCert != "" && *serverCert != "" && *serverKey != ""
}

func setupTLS(certFile, keyFile, CAFile string) (*tls.Config, error) {
	var err error
	tlsConfig := &tls.Config{}
	tlsConfig.Certificates = make([]tls.Certificate, 1)
	tlsConfig.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	b, err := ioutil.ReadFile(CAFile)
	if err != nil {
		return nil, err
	}
	ca := x509.NewCertPool()
	ok := ca.AppendCertsFromPEM([]byte(b))
	if !ok {
		return nil, fmt.Errorf("failed to parse root certificate: %q", CAFile)
	}
	tlsConfig.ClientCAs = ca
	return tlsConfig, nil
}
