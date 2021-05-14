bin/webserver:
	go build -o bin/webserver ./cmd/webserver

.PHONY: run
run: bin/webserver
	./bin/webserver

.PHONY: run-ssl
run-ssl: bin/webserver certs/server.pem
	./bin/webserver -ca-cert=certs/ca.pem \
		-server-cert=certs/server.pem \
		-server-key=certs/server-key.pem

certs/ca.pem:
	cfssl gencert -initca certs/ca-csr.json | cfssljson -bare ca
	mv *.pem *.csr ./certs/

certs/server.pem: certs/ca.pem
	cfssl gencert \
		-ca=certs/ca.pem \
		-ca-key=certs/ca-key.pem \
		-config=certs/ca-config.json \
		-profile=server \
		certs/server-csr.json | cfssljson -bare server
		mv *.pem *.csr ./certs/

.PHONY: clean
clean:
	rm -f bin/*
