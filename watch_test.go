package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"

	consulapi "github.com/hashicorp/consul/api"
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

func createTempDir(t *testing.T) string {
	tempDir, err := ioutil.TempDir("", "fsconsul_test")
	defer os.RemoveAll(tempDir)

	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	return tempDir
}

func writeToConsul(t *testing.T, prefix, key string) []byte {

	token := os.Getenv("TOKEN")
	dc := os.Getenv("DC")
	if dc == "" {
		dc = "dc1"
	}

	client := makeConsulClient(t)
	kv := client.KV()

	writeOptions := &consulapi.WriteOptions{Token: token, Datacenter: dc}

	// Delete all keys in the prefixed KV space
	if _, err := kv.DeleteTree(prefix, writeOptions); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Put a test KV
	encodedValue := make([]byte, base64.StdEncoding.EncodedLen(1024))
	base64.StdEncoding.Encode(encodedValue, createRandomBytes(1024))
	p := &consulapi.KVPair{Key: key, Flags: 42, Value: encodedValue}
	if _, err := kv.Put(p, writeOptions); err != nil {
		t.Fatalf("err: %v", err)
	}

	return encodedValue
}

func deleteKeyFromConsul(t *testing.T, key string) {

	token := os.Getenv("TOKEN")
	dc := os.Getenv("DC")
	if dc == "" {
		dc = "dc1"
	}

	client := makeConsulClient(t)
	kv := client.KV()

	writeOptions := &consulapi.WriteOptions{Token: token, Datacenter: dc}
	if _, err := kv.Delete(key, writeOptions); err != nil {
		t.Fatalf("err: %v", err)
	}
}

var configBlobs = []struct {
	json, key string
}{
	{
		`{
			"mappings" : [{
				"onchange": "date",
				"prefix": "simple_file"
			}]
		}`,
		"randomEntry",
	}, {
		`{
			"mappings" : [{
				"onchange": "date",
				"prefix": "nested/file"
			}]
		}`,
		"simple_file",
	}, {
		`{
			"mappings" : [{
				"onchange": "date",
				"prefix": "gotest/randombytes"
			}]
		}`,
		"entry",
	},
}

func TestConfigBlobs(t *testing.T) {

	for _, test := range configBlobs {

		tempDir := createTempDir(t)

		var config WatchConfig

		err := json.Unmarshal([]byte(test.json), &config)
		if err != nil {
			t.Fatalf("Failed to parse JSON due to %v", err)
		}

		key := config.Mappings[0].Prefix + "/" + test.key

		fmt.Println("Starting test with key", key)

		// Run the fsconsul listener in the background
		go func() {

			config.Mappings[0].Path = tempDir + "/"

			rvalue := watchAndExec(&config)
			if rvalue == -1 {
				t.Fatalf("Failed to run watchAndExec")
			}

			if config.Mappings[0].Path[len(config.Mappings[0].Path)-1] == 34 {
				t.Fatalf("Config path should have trailing spaces stripped")
			}

		}()

		encodedValue := writeToConsul(t, config.Mappings[0].Prefix, key)

		// Give ourselves a little bit of time for the watcher to read the file
		time.Sleep(100 * time.Millisecond)

		fileValue, err := ioutil.ReadFile(path.Join(tempDir, test.key))
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		if !bytes.Equal(encodedValue, fileValue) {
			t.Fatal("Unmatched values")
		}
	}
}

var deleteableConfigBlobs = []struct {
	json, key string
}{
	{
		`{
			"mappings" : [{
				"onchange": "date",
				"prefix": "simple_file"
			}]
		}`,
		"randomEntry",
	}, {
		`{
			"mappings" : [{
				"onchange": "date",
				"prefix": "nested/file"
			}]
		}`,
		"simple_file",
	}, {
		`{
			"mappings" : [{
				"onchange": "date",
				"prefix": "gotest/randombytes"
			}]
		}`,
		"entry",
	},
}

func TestConfigBlobsForDelete(t *testing.T) {

	for _, test := range deleteableConfigBlobs {

		tempDir := createTempDir(t)

		var config WatchConfig

		err := json.Unmarshal([]byte(test.json), &config)
		if err != nil {
			t.Fatalf("Failed to parse JSON due to %v", err)
		}

		key := config.Mappings[0].Prefix + "/" + test.key

		fmt.Println("Starting test with key", key)

		// Run the fsconsul listener in the background
		go func() {

			config.Mappings[0].Path = tempDir + "/"

			rvalue := watchAndExec(&config)
			if rvalue == -1 {
				t.Fatalf("Failed to run watchAndExec")
			}

			if config.Mappings[0].Path[len(config.Mappings[0].Path)-1] == 34 {
				t.Fatalf("Config path should have trailing spaces stripped")
			}

		}()

		encodedValue := writeToConsul(t, config.Mappings[0].Prefix, key)

		// Give ourselves a little bit of time for the watcher to read the file
		time.Sleep(100 * time.Millisecond)

		keyfilePath := path.Join(tempDir, test.key)

		fileValue, err := ioutil.ReadFile(keyfilePath)
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		if !bytes.Equal(encodedValue, fileValue) {
			t.Fatal("Unmatched values")
		}

		deleteKeyFromConsul(t, key)

		// Give ourselves a little bit of time for the watcher to delete the file
		time.Sleep(100 * time.Millisecond)

		if _, err := os.Stat(keyfilePath); os.IsExist(err) {
			t.Fatalf("Key file still exists even after delete")
		}
	}
}
var simpleConfigBlob = struct {
	json, key string
}{
	`{
		"mappings" : [{
			"onchange": "date",
			"prefix": "simple_file"
		}]
	}`,
	"killMe",
}

func countOpenFiles() int {
	out, err := exec.Command("/bin/sh", "-c", fmt.Sprintf("lsof -p %v | grep REG", os.Getpid())).Output()
	if err != nil {
		fmt.Println("Failed to get open file count due to ", err)
		return 100000
	}
	fmt.Println(string(out))
	lines := bytes.Count(out, []byte("\n"))
	return lines - 1
}

func TestAgainstLeaks(t *testing.T) {

	var config WatchConfig

	err := json.Unmarshal([]byte(simpleConfigBlob.json), &config)

	key := config.Mappings[0].Prefix + "/" + simpleConfigBlob.key

	tempDir := createTempDir(t)

	if err != nil {
		t.Fatalf("Failed to parse JSON due to %v", err)
	}

	// Run the fsconsul listener in the background
	go func() {

		config.Mappings[0].Path = tempDir + "/"

		rvalue := watchAndExec(&config)
		if rvalue == -1 {
			t.Fatalf("Failed to run watchAndExec")
		}

		if config.Mappings[0].Path[len(config.Mappings[0].Path)-1] == 34 {
			t.Fatalf("Config path should have trailing spaces stripped")
		}

	}()

	for i:= 0; i < 100; i++ {

		_ = writeToConsul(t, config.Mappings[0].Prefix, key)

		// Give ourselves a little bit of time for the watcher to read the file
		time.Sleep(100 * time.Millisecond)
	}

	deleteKeyFromConsul(t, key)

	openFileCount := countOpenFiles()

	// Validate that number of open files is not bananas.
	fmt.Printf("There are %d open files\n", openFileCount)
	if openFileCount > 10 {
		t.Fatalf("There are %d open files.  That's too damn high.", openFileCount)
	}
}
