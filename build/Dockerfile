## Build
FROM golang:1.22 AS build

WORKDIR /cmd

# Copy all files
COPY .. ./

# Download necessary Go modules
RUN go mod download

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/app -v ./cmd/cli

## Deploy
FROM scratch AS final

WORKDIR /

COPY --from=build /bin/app /app

EXPOSE 8080
EXPOSE 8090

ENTRYPOINT ["/app"]