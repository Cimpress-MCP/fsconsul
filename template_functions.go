package main

import (
	"encoding/base64"
	"fmt"
	gosecret "github.com/cimpress-mcp/gosecret/api"
)

func goEncryptFunc(keystore string) func(...string) (string, error) {
	return func(s ...string) (string, error) {
		dt, err := gosecret.ParseEncrytionTag(keystore, s...)
		if err != nil {
			fmt.Println("Unable to parse encryption tag", err)
			return "", err
		}

		return (fmt.Sprintf("{{goDecrypt \"%s\" \"%s\" \"%s\" \"%s\"}}",
			dt.AuthData,
			base64.StdEncoding.EncodeToString(dt.CipherText),
			base64.StdEncoding.EncodeToString(dt.InitVector),
			dt.KeyName)), nil
	}
}

func goDecryptFunc(keystore string) func(...string) (string, error) {
	return func(s ...string) (string, error) {
		plaintext, err := gosecret.ParseDecryptionTag(keystore, s...)
		if err != nil {
			fmt.Println("Unable to parse encryption tag", err)
			return "", err
		}

		return fmt.Sprintf("%s", plaintext), nil
	}
}
