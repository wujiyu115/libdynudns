\<DNYN\> for [`libdns`](https://github.com/libdns/libdns)
=======================

[![Go Reference](https://pkg.go.dev/badge/test.svg)](https://pkg.go.dev/github.com/wujiyu115/libdynudns)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for \<DNYN\>, allowing you to manage DNS records.

## Config examples

To use this module for the ACME DNS challenge, [configure the ACME issuer in your Caddy JSON](https://caddyserver.com/docs/json/apps/tls/automation/policies/issuer/acme/) like so:

```json
{
	"module": "acme",
	"challenges": {
		"dns": {
			"provider": {
				"name": "dynu",
				"api_token": "YOUR_PROVIDER_API_TOKEN"
				"proxy_url": "http://192.168.31.139:8118"
			}
		}
	}
}
```

or with the Caddyfile:

```
# globally
{
	acme_dns dynu ...
}
```

```
# one site
tls {
	dns dynu ...
}
```