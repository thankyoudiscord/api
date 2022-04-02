FROM golang:1.17.6-alpine AS builder
RUN apk add make git protoc
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
WORKDIR /usr/src/app
COPY . .
RUN git submodule init
RUN make build

FROM alpine AS runtime
COPY --from=builder /usr/src/app/birthday-backend /birthday-backend
ENTRYPOINT /birthday-backend
