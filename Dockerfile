FROM golang:1-bookworm AS build

WORKDIR /src/
# Pull in deps first to ease cache
COPY go.mod go.sum ./
RUN go mod download

COPY . ./

# Use linkerflags to strip DWARF info
RUN CGO_ENABLED=0 go build -ldflags="-s -w"  -o /bin/switchhost

# Now create the runtime container from python base for nsz

FROM python:3.12-slim-bookworm

WORKDIR /switchhost/
RUN  pip3 install nsz==4.5.0 && apt-get update && apt-get install -y curl && apt-get purge
COPY --from=build /bin/switchhost ./switchhost

# Run healthcheck against the web ui
HEALTHCHECK CMD curl --fail http://localhost:8080/healthcheck || exit 1   

ENTRYPOINT ["/switchhost/switchhost", "--config","/data/config.json","--keys","/data/prod.keys","--noCUI"]
