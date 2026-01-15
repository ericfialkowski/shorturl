# Code Build
FROM golang:1.25-alpine3.23 AS  build

WORKDIR /go/src/app
COPY . .
RUN go build -v

# Create Deployable
FROM scratch
COPY --from=build /go/src/app/*.html /
COPY --from=build /go/src/app/shorturl /shorturl

EXPOSE 8800

ENV mongo_uri=""
ENV logrequests=false
ENV keygrowretries=10
ENV startingkeysize=2

CMD ["/shorturl"]
