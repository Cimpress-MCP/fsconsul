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
		&keystore, "token", "",
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

	config := WatchConfig{
		ConsulAddr: consulAddr,
		ConsulDC:   consulDC,
		OnChange:   onChange,
		Path:       args[1],
		Prefix:     args[0],
		Keystore:   keystore,
		Token:      token,
	}
	result, err := watchAndExec(&config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		return 111
	}

	return result
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
