# docker-ce needs to be 17.05 or later.
FROM golang:alpine as go_builder

COPY vendor/ $GOPATH/src/
COPY *.go /src/
WORKDIR /src/
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o webhookd .

FROM scratch
COPY --from=go_builder   $GOPATH/src/webhookd .
ENTRYPOINT ["/webhookd"]
