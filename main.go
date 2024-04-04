/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"fmt"
	"os"

	"github.com/Emyrk/screeps-watcher/cmd"
)

func main() {
	err := cmd.New().RootCmd().Invoke().WithOS().Run()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
	}
}
