.PHONY: build vet fmt test race run docker-build docker-run clean

BINARY := marine-farm-environment-service
IMAGE  := marine-farm-environment-service:latest

build:
	go build -o $(BINARY) .

vet:
	go vet ./...

fmt:
	gofmt -w .

test:
	go test ./...

race:
	go test -race ./...

run:
	go run .

docker-build:
	docker build -t $(IMAGE) .

docker-run:
	docker run --rm -p 8080:8080 $(IMAGE)

clean:
	rm -f $(BINARY)
	rm -rf data
