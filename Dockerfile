FROM ${ARCH}golang:1.15-alpine AS build_base
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
RUN CGO_ENABLED=0 go test -v ./cloud

# Build the Go app
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o ./out/cost-report .

# Start fresh from a smaller image
# kics-scan ignore-line
FROM scratch

COPY --from=build_base /etc/passwd /etc/passwd
USER scratchuser
# kics-scan ignore-line
COPY --from=build_base /tmp/cost-report/out/cost-report /app/cost-report
COPY --from=build_base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# This container exposes port 8080 to the outside world
EXPOSE 8080

# Run the binary program produced by `go install`
CMD ["/app/cost-report"]