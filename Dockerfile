FROM golang:1.17.6-alpine AS builder
RUN apk add make git
WORKDIR /usr/src/app
COPY . .
RUN make build

FROM alpine AS runtime
COPY --from=builder /usr/src/app/birthday-backend /birthday-backend
ENTRYPOINT /birthday-backend
