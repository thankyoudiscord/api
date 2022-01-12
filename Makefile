OUTPUT := birthday-backend

$(OUTPUT): *.go
	go build -o $(OUTPUT) *.go

build: $(OUTPUT)

.PHONY: clean
clean:
	@go clean
	@rm -f $(OUTPUT)

.PHONY: dev
dev:
	@go run *.go
