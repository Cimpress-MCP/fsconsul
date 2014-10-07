package main

import (
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"time"

	"github.com/armed/mkdirp"
	"github.com/armon/consul-api"

	"github.com/ryanbreen/gosecret"
)

// Configuration for watches.
type WatchConfig struct {
	ConsulAddr string
	ConsulDC   string
	OnChange   []string
	Prefix     string
	Path       string
	Keystore   string
	Token      string
}

// Connects to Consul and watches a given K/V prefix and uses that to
// write to the filesystem.
func watchAndExec(config *WatchConfig) (int, error) {
	kvConfig := consulapi.DefaultConfig()
	kvConfig.Address = config.ConsulAddr
	kvConfig.Datacenter = config.ConsulDC

	client, err := consulapi.NewClient(kvConfig)
	if err != nil {
		return 0, err
	}

	// If the config path is lacking a trailing separator, add it.
	if config.Path[len(config.Path)-1] != os.PathSeparator {
		config.Path += string(os.PathSeparator)
	}

	isWindows := os.PathSeparator != '/'

	// Start the watcher goroutine that watches for changes in the
	// K/V and notifies us on a channel.
	errCh := make(chan error, 1)
	pairCh := make(chan consulapi.KVPairs)
	quitCh := make(chan struct{})
	defer close(quitCh)
	go watch(
		client, config.Prefix, config.Path, config.Token, pairCh, errCh, quitCh)

	var env map[string]string
	for {
		var pairs consulapi.KVPairs

		// Wait for new pairs to come on our channel or an error
		// to occur.
		select {
		case pairs = <-pairCh:
		case err := <-errCh:
			return 0, err
		}

		newEnv := make(map[string]string)
		for _, pair := range pairs {
			k := strings.TrimPrefix(pair.Key, config.Prefix)
			k = strings.TrimLeft(k, "/")
			newEnv[k] = string(pair.Value)
		}

		// If the variables didn't actually change,
		// then don't do anything.
		if reflect.DeepEqual(env, newEnv) {
			continue
		}

		// Blocked by KV().List bug, will not rebuild unless all keys deleted ATM, so we
		// are disabling deletes.
		// TODO: Make this more granular: only delete keys that were deleted based on the
		// tombstones contained in a future version of Consul.
		/**
		if _, err := os.Stat(config.Path); err == nil {
			os.RemoveAll(config.Path)
			mkdirp.Mk(config.Path, 0777)
			fmt.Println("Tree rebuild triggered")
		}
		**/

		// Replace the env so we can detect future changes
		env = newEnv

		//fmt.Println(env)

		// Write the updated keys to the filesystem at the specified path
		for k, v := range newEnv {
			// Write file to disk
			keyfile := fmt.Sprintf("%s%s", config.Path, k)

			// if Windows, replace / with windows path delimiter
			if isWindows {
				keyfile = strings.Replace(keyfile, "/", "\\", -1)
				// mkdirp the file's path
				err := mkdirp.Mk(keyfile[:strings.LastIndex(keyfile, "\\")], 0777)
				if err != nil {
					fmt.Println("Failed to create parent directory for key", err)
				}
			} else {
				// mkdirp the file's path
				err := mkdirp.Mk(keyfile[:strings.LastIndex(keyfile, "/")], 0777)
				if err != nil {
					fmt.Println("Failed to create parent directory for key", err)
				}
			}

			f, err := os.Create(keyfile)
			if err != nil {
				fmt.Printf("Failed to create file %s due to %s\n", keyfile, err)
				continue
			}

			defer f.Close()

			fmt.Println("Input value length:", len(v))

			decryptedValue, err := gosecret.DecryptTags([]byte(v), config.Keystore)
			if err != nil {
				fmt.Println("Failed to decrypt value due to", err)
				decryptedValue = []byte(v)
			}

			fmt.Println("Output length:", len(decryptedValue))

			wrote, err := f.Write(decryptedValue)
			if err != nil {
				fmt.Printf("Failed to write to file %s due to %s\n", keyfile, err)
				continue
			}

			fmt.Printf("Successfully wrote %d bytes to %s\n", wrote, keyfile)

			f.Sync()
		}

		// Configuration changed, run our onchange command, if one was specified.
		if config.OnChange != nil {
			var cmd = exec.Command(config.OnChange[0], config.OnChange[1:]...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Start()
			if err != nil {
				return 111, err
			}
		}
	}
}

func watch(
	client *consulapi.Client,
	prefix string,
	path string,
	token string,
	pairCh chan<- consulapi.KVPairs,
	errCh chan<- error,
	quitCh <-chan struct{}) {

	// Create the root for KVs, if necessary
	mkdirp.Mk(path, 0777)

	// Get the initial list of k/v pairs. We don't do a retryableList
	// here because we want a fast fail if the initial request fails.
	opts := &consulapi.QueryOptions{Token: token}
	pairs, meta, err := client.KV().List(prefix, opts)
	if err != nil {
		errCh <- err
		return
	}

	// Send the initial list out right away
	pairCh <- pairs

	// Loop forever (or until quitCh is closed) and watch the keys
	// for changes.
	curIndex := meta.LastIndex
	for {
		select {
		case <-quitCh:
			return
		default:
		}

		pairs, meta, err = retryableList(
			func() (consulapi.KVPairs, *consulapi.QueryMeta, error) {
				opts = &consulapi.QueryOptions{WaitIndex: curIndex, Token: token}
				return client.KV().List(prefix, opts)
			})

		if meta == nil {
			// This happens when the connection to the consul agent dies.  Build in a retry by looping after a delay.
			fmt.Println("Error communicating with consul agent.")
			time.Sleep(time.Duration(5) * time.Second)
		} else {
			pairCh <- pairs
			fmt.Printf("curIndex: %d lastIndex: %d\n", curIndex, meta.LastIndex)
			curIndex = meta.LastIndex
		}
	}
}

// This function is able to call KV listing functions and retry them.
// We want to retry if there are errors because it is safe (GET request),
// and erroring early is MUCH more costly than retrying over time and
// delaying the configuration propagation.
func retryableList(f func() (consulapi.KVPairs, *consulapi.QueryMeta, error)) (consulapi.KVPairs, *consulapi.QueryMeta, error) {
	i := 0
	for {
		p, m, e := f()
		if e != nil {
			if i >= 3 {
				return nil, nil, e
			}

			i++

			// Reasonably arbitrary sleep to just try again... It is
			// a GET request so this is safe.
			time.Sleep(time.Duration(i*2) * time.Second)
		}

		return p, m, e
	}
}
