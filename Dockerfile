FROM golang:1.17.5-alpine3.15 as builder

WORKDIR /go/src/hass-shooter

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build


FROM alpine:3.15

RUN apk update && apk add chromium imagemagick

WORKDIR /app

COPY --from=builder /go/src/hass-shooter/hass-shooter hass-shooter

ENTRYPOINT ["/app/hass-shooter"]
CMD ["-c", "/data/options.json"]
