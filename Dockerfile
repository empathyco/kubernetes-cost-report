ARG ARCH=

FROM ${ARCH}golang:1.16.12-alpine3.15

# Set necessary environmet variables needed for our image
ENV GO111MODULE=on \
    CGO_ENABLED=0 
    
WORKDIR /build
COPY . .
COPY go.mod .
COPY go.sum . 
RUN go mod download
RUN go build -o main main.go
WORKDIR /dist 

RUN cp /build/main .
EXPOSE 8080 

CMD ["/dist/main"]