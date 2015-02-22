# fsconsul

fsconsul writes data to the filesystem by reading it from [Consul's K/V store](http://www.consul.io).

fsconsul allows configuration data to be placed on disk without a consuming application
needing to know about the existence of Consul. This makes it especially easy to configure
applications throughout all your environments: development, testing, production, etc.  On
any change, fsconsul will run the provided command, so you can perform any additional actions
(restarting your application, for example) necessary.

fsconsul is a port of [envconsul](https://github.com/hashicorp/envconsul) which in turn was inspired by [envdir](http://cr.yp.to/daemontools/envdir.html)
in its simplicity, name, and function.

## Download & Usage

To install fsconsul, clone this repo into your go workspace and do a `go install`.

## Configuration

`fsconsul` can be configured entirely by command-line switches, but for more complex cases, you may wish to provide the path to a config JSON file as the -configFile switch.  The format of the JSON file is:

```
{
	"consul" : {
		"addr": "127.0.0.1:8500"
		"dc": "dc1",
		"token" : "my-reader-token"
	},
	"mappings" : [{
		"onchange": "service restart app1",
		"prefix": "/myteam/dev/app1/config/",
		"path": "/etc/app1/",
		"keystore": "/var/lib/encryption_keys"
	},{
		"onchange": "service restart app2",
		"prefix": "/myteam/dev/app2/config/",
		"path": "/etc/app2/",
		"keystore": "/var/app2/encryption_keys"
	}]
}

``` 

Run `fsconsul` to see the usage help:

```

$ fsconsul
Usage: fsconsul [options] prefix path onchange

  Write files to the specified locations on the local system by reading K/Vs
  from Consul's K/V store with the given prefixes and executing a program on
  any change.  Prefixes and paths must be pipe-delimited if provided as
  command-line switches.

Options:

  -addr="": consul HTTP API address with port
  -configFile="": json file containing all configuration (if this is provided, all other config is ignored)
  -dc="": consul datacenter, uses local if blank
  -keystore="": directory of keys used for decryption
  -once=false: run once and exit
  -token="": token to use for ACL access
```

## CI

Builds are automatically run by Travis on any push or pull request.

![Travis Status](https://travis-ci.org/Cimpress-MCP/fsconsul.svg?branch=master)

##

Tagged builds are automatically published to bintray for OS X, Linux, and Windows.

[ ![Download](https://api.bintray.com/packages/cimpress-mcp/Go/fsconsul/images/download.svg) ](https://bintray.com/cimpress-mcp/Go/fsconsul/_latestVersion#files)

## TODO

* Once Consul 0.5 is out, support file deletes.
