package main

import (
	"crypto/rand"
	"fmt"
)

func genHexColor() (color []byte, err error) {
	c := 3
	b := make([]byte, c)
	_, err = rand.Read(b)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	return b, nil
}
