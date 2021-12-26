/*
hass-shooter is a Home Assistant screenshot capture web server suitable for
e-ink displays.

Usage:

	hass-shooter [flags]

The flags are:

	-c string
		configuration file (default "/data/options.json")

hass-shooter depends on Chromium and ImageMagick, so they must be installed
first.

The configuration file is a JSON file with the following format:

	{
	  "hass_base_url": "https://example.com",
	  "hass_token": "ACCESS_TOKEN",
	  "hass_pages": [
	    {
	      "path": "/lovelace/default_view",
	      "scale": 1
	    }
	  ],
	  "width": 480,
	  "height": 800,
	  "rotation": 0,
	  "listen_addr": ":8000",
	  "refresh_time": 60,
	  "min_idle_time": 5,
	  "timeout": 60
	}

The following configuration parameters are supported:

"hass_base_url" is the URL of the Home Assistant server.

"hass_token" is a Home Assistant long-lived access token. More information:
https://developers.home-assistant.io/docs/auth_api/#long-lived-access-token

"hass_pages" contains the Home Assistant pages to capture. "path" is the URL
path of the page. The full captured URL is hass_base_url + hass_pages[i].path.
"scale" is the scale factor used to capture the page.

"width" is the width of the generate image (usually, the width of the e-ink display).

"height" is the height of the generate image (usually, the height of the e-ink
display).

"rotation" is the rotation in degrees applied to the resulting images.

"listen_addr" is the HTTP service address to listen for incoming requests.

"refresh_time" is the time in seconds between screenshots.

"min_idle_time" is the minimum time in seconds without requests to consider a page to
be loaded.

"timeout" is the timeout in seconds used by the headless browser.
*/
package main
