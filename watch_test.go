package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/armon/consul-api"
)

func createRandomBytes(length int) []byte {
	random_bytes := make([]byte, length)
	rand.Read(random_bytes)
	return random_bytes
}

func makeConsulClient(t *testing.T) *consulapi.Client {
	conf := consulapi.DefaultConfig()
	client, err := consulapi.NewClient(conf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	return client
}

func TestAddFile(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "fsconsul_test")
	defer os.RemoveAll(tempDir)

	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	client := makeConsulClient(t)
	kv := client.KV()

	key := "gotest/randombytes/entry"

	token := os.Getenv("TOKEN")
	dc := os.Getenv("DC")
	if dc == "" {
		dc = "dc1"
	}

	writeOptions := &consulapi.WriteOptions{Token: token, Datacenter: dc}

	// Delete all keys in the "gotest" KV space
	if _, err := kv.DeleteTree("gotest", writeOptions); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Run the fsconsul listener in the background
	go func() {

		//consulConfig := 

		config := WatchConfig{
			Consul: ConsulConfig{
				Addr: consulapi.DefaultConfig().Address,
				DC:   dc,
				Token:      token,
			},
			Mappings: make([]MappingConfig, 1),
		}

		config.Mappings[0] = MappingConfig {
			Path:       tempDir + "/",
			Prefix:     "gotest",
		}

		rvalue := watchAndExec(&config)
		if rvalue == -1 {
			t.Fatalf("Failed to run watchAndExec")
		}

		if config.Mappings[0].Path[len(config.Mappings[0].Path)-1] == 34 {
			t.Fatalf("Config path should have trailing spaces stripped")
		}

	}()

	// Put a test KV
	encodedValue := make([]byte, base64.StdEncoding.EncodedLen(1024))
	base64.StdEncoding.Encode(encodedValue, createRandomBytes(1024))
	p := &consulapi.KVPair{Key: key, Flags: 42, Value: encodedValue}
	if _, err := kv.Put(p, writeOptions); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Give ourselves a little bit of time for the watcher to read the file
	time.Sleep(100 * time.Millisecond)

	fileValue, err := ioutil.ReadFile(path.Join(tempDir, "randombytes", "entry"))
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if !bytes.Equal(encodedValue, fileValue) {
		t.Fatal("Unmatched values")
	}
}
