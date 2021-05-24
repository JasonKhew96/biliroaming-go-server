FROM golang:alpine AS builder

WORKDIR /tmp/builder/

COPY go.mod go.sum ./
RUN go mod download -x

COPY . .
RUN env CGO_ENABLED=0 go build -v -o /tmp/server

FROM gcr.io/distroless/static:latest
WORKDIR /runner
COPY --from=builder /tmp/server .
CMD [ "./server" ]