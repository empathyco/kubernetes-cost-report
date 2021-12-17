FROM ${ARCH}golang:1.15-alpine AS build_base
ENV CI=docker
RUN apk add --no-cache git

# Set the Current Working Directory inside the container
WORKDIR /tmp/cost-report

# We want to populate the module cache based on the go.{mod,sum} files.
COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

# Unit tests
RUN CGO_ENABLED=0 go test -v

# Build the Go app
RUN go build -o ./out/cost-report .

# Start fresh from a smaller image
FROM alpine:3.9 
RUN apk add ca-certificates

COPY --from=build_base /tmp/cost-report/out/cost-report /app/cost-report

# This container exposes port 8080 to the outside world
EXPOSE 8080

# Run the binary program produced by `go install`
CMD ["/app/cost-report"]