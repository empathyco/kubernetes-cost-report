ARG ARCH=

FROM ${ARCH}golang:1.15-alpine

# Set necessary environmet variables needed for our image
ENV GO111MODULE=on \
    CGO_ENABLED=0 
    
WORKDIR /build
COPY go.mod .
COPY go.sum . 
RUN go mod download
COPY . .
RUN go build -o main main.go
RUN ls -ltr
WORKDIR /dist 

RUN cp /build/main .
EXPOSE 8080 

CMD ["/dist/main"]