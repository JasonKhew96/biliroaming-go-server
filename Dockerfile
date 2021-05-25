FROM golang:alpine AS builder

WORKDIR /tmp/builder/

COPY go.mod go.sum ./
RUN go mod download -x

COPY . .
ENV CGO_ENABLED=0
RUN go build -v -o /tmp/server

FROM gcr.io/distroless/static:latest
WORKDIR /runner
COPY --from=builder /tmp/server .
CMD [ "./server" ]