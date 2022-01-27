FROM ${ARCH}golang:1.17-alpine AS build_base
ENV CI=docker
RUN apk add --no-cache "git>2.32.0" "ca-certificates>20211220" && \
    addgroup -S scratchuser && adduser -S scratchuser -G scratchuser

USER scratchuser
# Set the Current Working Directory inside the container
WORKDIR /tmp/cost-report

# We want to populate the module cache based on the go.{mod,sum} files.
COPY go.* .
# COPY go.sum .

RUN go mod download

COPY . .

# Unit tests
# RUN CGO_ENABLED=0 go test -v ./pkg

# Build the Go app
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o ./out/co2 .

# Start fresh from a smaller image
# kics-scan ignore-line
FROM scratch

COPY --from=build_base /etc/passwd /etc/passwd
# kics-scan ignore-line
COPY --from=build_base /tmp/cost-report/out/co2 /app/co2
COPY --from=build_base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
USER scratchuser

# This container exposes port 8080 to the outside world
EXPOSE 8080

# Run the binary program produced by `go install`
CMD ["/app/co2"]
