package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	os.Exit(realMain())
}

func realMain() int {
	var consulAddr string
	var consulDC string
	var keystore string
	var token string
	var configFile string
	flag.Usage = usage
	flag.StringVar(
		&consulAddr, "addr", "127.0.0.1:8500",
		"consul HTTP API address with port")
	flag.StringVar(
		&consulDC, "dc", "",
		"consul datacenter, uses local if blank")
	flag.StringVar(
		&keystore, "keystore", "",
		"directory of keys used for decryption")
	flag.StringVar(
		&token, "token", "",
		"token to use for ACL access")
	flag.StringVar(
		&configFile, "configFile", "",
		"json file containing all configuration")
	flag.Parse()
	if flag.NArg() < 2 {
		flag.Usage()
		return 1
	}

	args := flag.Args()

	var onChange []string
	if len(args) > 2 {
		onChange = args[2:]
	}

	// Check whether multiple paths / prefixes are specified
	var prefixes = strings.Split(args[0], "|")
	var paths = strings.Split(args[1], "|")

	if len(prefixes) != len(paths) {
		fmt.Fprintf(os.Stderr, "Error: There must be an identical number of prefixes and paths.\n")
		return 1
	}

	config := WatchConfig{
		Consul: ConsulConfig {
			Addr: consulAddr,
			DC: consulDC,
			Token: token,
		},
		Mappings: make([]MappingConfig, len(prefixes)),
	}

	for i := 0; i < len(prefixes); i++ {
		config.Mappings[i] = MappingConfig{
			Prefix:     prefixes[i],
			Path:       paths[i],
			Keystore:   keystore,
			OnChange:   onChange,
		}
	}

	return watchAndExec(&config)
}

func usage() {
	cmd := filepath.Base(os.Args[0])
	fmt.Fprintf(os.Stderr, strings.TrimSpace(helpText)+"\n\n", cmd)
	flag.PrintDefaults()
}

const helpText = `
Usage: %s [options] prefix path onchange

  Write files to the specified location on the local system by reading K/V
  from Consul's K/V store with the given prefix and execute a program on
  any change.

Options:
`
