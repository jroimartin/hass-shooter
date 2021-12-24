# hass-shooter

[![Go Reference](https://pkg.go.dev/badge/github.com/jroimartin/hass-shooter.svg)](https://pkg.go.dev/github.com/jroimartin/hass-shooter)

hass-shooter is a Home Assistant screenshot capture web server suitable for
e-ink displays.

## Dependencies

- [Chromium](https://www.chromium.org/)
- [ImageMagick](https://imagemagick.org/)

## Installation

```
go install github.com/jroimartin/hass-shooter@latest
```

## Usage

Please, see command documentation:

- Local: `go doc`
- Online: [pkg.go.dev](https://pkg.go.dev/github.com/jroimartin/hass-shooter)

## Docker

Build the hass-shooter Docker image:

```
docker build -t hass-shooter .
```

Run it:

```
docker run -ti --rm -p 8000:8000 -v /config/path:/data \
    hass-shooter -c /data/options.json
```
