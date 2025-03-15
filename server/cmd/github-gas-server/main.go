package main

import (
	"fmt"
	"os"

	"github.com/rubiojr/gas/internal/mcpserver"
)

func main() {
	s, err := mcpserver.New()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	s.ServeStdio()
}
