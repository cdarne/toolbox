# Toolbox

## Requirements

Requires `cfssl` and `cfssljson` to generate the certificates
```bash
$ go get github.com/cloudflare/cfssl/cmd/cfssl
$ go get github.com/cloudflare/cfssl/cmd/cfssljson
```

## `webserver`

A simple web server to test your HTTP clients. It gracefully stops on Ctrl+C, it can optionaly handle TLS and also logs the state of the underlying TCP connection (to track persisting connections).

### Usage

```bash
# HTTP
$ make clean && make run
# or manually
$ go build -o bin/webserver ./cmd/webserver
$ ./bin/webserver

# then you can do
$ curl -vv http://localhost:1984 -X POST http://localhost:1984
# or
$ ab -n 100 -c 10 -k -p ~/sample.json -T "application/json; charset=utf-8" http://localhost:1984

# HTTPS
$ make clean && make run-ssl
# or manually
# generates the CA and server key pairs
$ make certs/server.pem
$ go build -o bin/webserver ./cmd/webserver
$ ./bin/webserver -ca-cert=certs/ca.pem -server-cert=certs/server.pem -server-key=certs/server-key.pem

# then you can do
curl --http1.1 --cacert certs/ca.pem -vv https://127.0.0.1:1984 -X POST https://127.0.0.1:1984
# or
ab -n 100 -c 10 -k -p ~/sample.json -T "application/json; charset=utf-8" https://127.0.0.1:1984
```

### Note

When using TLS, don't forget to add the CA certificate (`certs/ca.pem`) to your client to be able to validate the server's cert.
