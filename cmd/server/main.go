package main

import (
	"fmt"
	"sshpong/internal/netwrk"
)

func main() {
	fmt.Println("Starting sshpong server!")

	netwrk.Listen()
}
