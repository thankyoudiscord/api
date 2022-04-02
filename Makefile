OUTPUT := birthday-backend
MAIN := cmd/thankyoudiscord/main.go
PROTOS_DIR := protos
PROTOS_OUT_DIR := pkg/protos

build: $(OUTPUT)

$(OUTPUT): protos $(wildcard */*/*.go)
	go build -o $(OUTPUT) $(MAIN)

protos: get_protos $(wildcard $(PROTOS_OUT_DIR)/*.pb.go)
	mkdir -p "$(PROTOS_OUT_DIR)"
	protoc \
		-I="$(PROTOS_DIR)" \
		--go_out="." \
		--go-grpc_out="." \
		$(PROTOS_DIR)/*.proto

get_protos:
	git submodule update

clean:
	go clean
	rm -f $(OUTPUT)
	rm -f $(PROTOS_OUT_DIR)/*.go

dev:
	@go run *.go

.PHONY: protos get_protos clean deps dev
