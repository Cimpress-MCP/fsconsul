package main

import (
	"flag"
	"fmt"
	"os"
	"os/user"
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
	flag.Parse()
	if flag.NArg() < 2 {
		flag.Usage()
		return 1
	}

	usr, err := user.Current()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		return 112
	}
	fmt.Printf("fsconsul root path: %s%sfsconsul\n", usr.HomeDir, string(os.PathSeparator)) 

	args := flag.Args()

	localPath := fmt.Sprintf("%s%sfsconsul%s", usr.HomeDir, string(os.PathSeparator), args[1]) 

	config := WatchConfig{
		ConsulAddr: consulAddr,
		ConsulDC:   consulDC,
		OnChange:   args[2:],
		Path:       localPath,
		Prefix:     args[0],
		Keystore:   keystore,
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
