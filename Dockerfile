# Stage 1 Build
FROM golang:stretch as build

WORKDIR /go/src/app
COPY * .
RUN go build -v

# Stage 2 Final
FROM debian:stretch-slim as final
COPY --from=build /go/src/app/shorturl /shorturl

EXPOSE 8800

CMD ["/shorturl"]
