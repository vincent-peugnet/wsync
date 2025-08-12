package main

import "fmt"

var version string = "unknown" // set by linker at compile time

func Version() {
	fmt.Println("wsync version", version)
}
