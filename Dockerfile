FROM golang:bullseye AS builder

WORKDIR /tmp/builder/

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ENV CGO_ENABLED=0
RUN go build -o /tmp/server

FROM gcr.io/distroless/static:latest
WORKDIR /runner
COPY sql sql
COPY --from=builder /tmp/server .
CMD [ "./server", "-config", "config/config.yml"]