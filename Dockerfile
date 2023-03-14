# Code Build
FROM golang:1.20-alpine3.17 as  build

WORKDIR /go/src/app
COPY . .
RUN go build -v

# Create Deployable
FROM alpine:3.17 as final
COPY --from=build /go/src/app/*.html /
COPY --from=build /go/src/app/shorturl /shorturl

EXPOSE 8800

ENV mongo_uri ""
ENV logrequests false
ENV keygrowretries 10
ENV startingkeysize 2

CMD ["/shorturl"]
