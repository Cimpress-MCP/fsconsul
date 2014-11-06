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

	returnCodes := make(chan int)

	// Fork a separate goroutine for each prefix/path pair
	for i := 0; i < len(prefixes); i++ {
		go func(config WatchConfig) {

			fmt.Println("Got watch config %s", config)

			returnCode, err := watchAndExec(&config)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			}

			returnCodes <- returnCode
		}(WatchConfig{
			ConsulAddr: consulAddr,
			ConsulDC:   consulDC,
			OnChange:   onChange,
			Prefix:     prefixes[i],
			Path:       paths[i],
			Keystore:   keystore,
			Token:      token,
		})
	}

	// Wait for completion of all forked go routines
	for i := 0; i < len(prefixes); i++ {
		fmt.Println(<-returnCodes)
	}

	return 0
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
