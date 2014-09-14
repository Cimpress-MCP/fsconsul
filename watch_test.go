package main

import (
	"fmt"
	"testing"
	"io/ioutil"
	"os"

	"github.com/armon/consul-api"
)

func makeConsulClient(t *testing.T) *consulapi.Client {
	conf := consulapi.DefaultConfig()
	client, err := consulapi.NewClient(conf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	return client
}

func TestAddFile(t* testing.T) {
	tempDir, err := ioutil.TempDir("", "fsconsul_test")

	fmt.Println("Created temp dir ", tempDir)

	if (err != nil) {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	client := makeConsulClient(t)
	fmt.Println("Created client ", client)

	config := WatchConfig{
		ConsulAddr: consulapi.DefaultConfig().Address,
		ConsulDC:   "dc1",
		OnChange:   []string{ "echo", "\"done\"" },
		Path:       tempDir,
		Prefix:     "test",
	}

	result, err := watchAndExec(&config)
	if err != nil {
		t.Fatalf("Failed to run watchAndExec: %v")
	}

	fmt.Println("result: ", result)

	err = os.RemoveAll(tempDir)
	if (err != nil) {
		t.Fatalf("Failed to clear temp dir: %v")
	}
}
