# check_log

## What is this?

A re-implementation of the original nagios check_log script
which has some implementation dependent issues with larger files.

This script aims to be higly scaleable, both in terms of file size as
well as CPU cores.

### Goals

* Minimize resource usage
* Handle large files
* Scale to many cores
* Handle the archetypes command line options

## Building

1. Install [go](http://golang.org/doc/install)

2. Compile check_log

  git clone git@github.com:gittex/nagios-check-log.git
  cd nagios-check-log
  go build

License
=======

Please see the LICENSE file.

