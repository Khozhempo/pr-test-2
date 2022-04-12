build:
	go get golang.org/x/net/html
#	GOOS=linux GOARCH=amd64 go build
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build
	cp /etc/ssl/certs/ca-certificates.crt ./
	docker build -t pr-test-2 -f Dockerfile .

run:
	docker run -p 8080:8080 pr-test-2
