package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("usage: %s shared-secret username [expires]\n", os.Args[0])
		return
	}

	secret := os.Args[1]
	username := os.Args[2]
	exp := time.Now().Add(7 * 24 * time.Hour) // 1 Week

	if len(os.Args) >= 4 {
		if ttl, err := ParseDuration(os.Args[3]); err == nil {
			exp = time.Now().Add(ttl)
		} else if exp, err = time.Parse("2006-01-02 15:04:05", os.Args[3]); err != nil {
			panic(err)
		}
	}

	user := fmt.Sprintf("%d:%s", exp.Unix(), username)

	digest := hmac.New(sha1.New, []byte(secret))
	digest.Write([]byte(user))

	passRaw := digest.Sum(nil)
	pass := base64.StdEncoding.EncodeToString(passRaw)

	fmt.Printf("Username: %s\n", user)
	fmt.Printf("Password: %s\n", pass)
}
