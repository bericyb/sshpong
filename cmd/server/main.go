package main

import (
	"fmt"
	"sshpong/internal/netwrk"
)

var exit chan bool

func main() {
	fmt.Println("Starting sshpong server!")

	netwrk.Listen()

	_ = <-exit
}
