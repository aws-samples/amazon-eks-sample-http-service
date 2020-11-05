FROM golang:1.15.4 as build

ENV GOPROXY=direct CGO_ENABLED=0

RUN mkdir /build
COPY go.mod go.sum main.go /build/
WORKDIR /build
RUN go mod download
RUN go build -o server

FROM scratch
COPY --from=build /build/server /server
COPY --from=build /etc/ssl/certs/ /etc/ssl/certs/
COPY html/ /html/

# Standard HTTP port
EXPOSE 8080
# HTTP wrapped in PROXY protocol
EXPOSE 9080

USER 101

ENTRYPOINT ["/server"]
