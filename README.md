# hass-shooter

[![Go Reference](https://pkg.go.dev/badge/github.com/jroimartin/hass-shooter.svg)](https://pkg.go.dev/github.com/jroimartin/hass-shooter)

hass-shooter is a Home Assistant screenshot capture web server suitable for
e-ink displays.

## Docs

Please, see the [command docs](https://pkg.go.dev/github.com/jroimartin/hass-shooter).

## Dependencies

- [Chromium](https://www.chromium.org/)
- [ImageMagick](https://imagemagick.org/)

## Installation

### Native

Install the hass-shooter command:

```
go install github.com/jroimartin/hass-shooter@latest
```

### Home Assistant add-on

hass-shooter is available as a Home Assistant add-on. Its installation and
configuration is similar to any other add-on. From your Home Assistant server:

1. Add this add-ons repository: https://github.com/jroimartin/hass-addons
2. Install the add-on "HASS Shooter".
3. Configure the add-on parameters from its Configuration tab.

### Docker

Build the hass-shooter Docker image:

```
docker build -t hass-shooter .
```

Run it:

```
docker run -ti --rm -p 8000:8000 -v /config/path:/data \
    hass-shooter -c /data/options.json
```
