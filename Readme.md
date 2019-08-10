# Retag

![Travis status badge](https://travis-ci.org/hugoShaka/retag.svg?branch=master)
![For the badge: made with go](https://forthebadge.com/images/badges/made-with-go.svg)
![For the badge: contains technical debt](https://forthebadge.com/images/badges/contains-technical-debt.svg)

A small binary to tag and move docker images **between repositories** of the same registry without pulling them.

Useful for CI jobs when you don't have locally the image.
This only works with a registry v2 hosting v2 images.

## Usage

```
Usage of ./retag:
  -debug
        sets verbosity level to debug
  -insecure
        use http instead of https
  -pass string
        specify auth password
  -user string
        specify auth user

Examples:
./retag -insecure nohttps.com/foo/bar/image:v1 nohttps.com/baz/prod:v5
./retag registry.com/img registry.com/hello:v9
./retag -user foo -password bar registry.com/img registry.com/hello:v9
```
