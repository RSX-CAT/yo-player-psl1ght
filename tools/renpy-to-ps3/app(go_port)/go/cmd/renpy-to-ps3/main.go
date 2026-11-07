package main

import (
	"os"

	"renpy-to-ps3/renpy"
)

func main() {
	os.Exit(renpy.Run(os.Args[1:]))
}
