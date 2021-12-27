FROM golang:1.17-alpine AS build

WORKDIR /src/
# Pull in deps first to ease cache
COPY go.mod go.sum ./
RUN go mod download

COPY . ./
# Use linkerflags to strip DWARF info
RUN CGO_ENABLED=0 go build -ldflags="-s -w"  -o /bin/switchhost

FROM alpine:3.15
WORKDIR /switchhost/
ENTRYPOINT ["/switchhost/switchhost"]