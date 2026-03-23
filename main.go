package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/danielbrandenburg/curlman/cmd"
)

func main() {
	if err := cmd.NewRootCmd().Execute(); err != nil {
		if !errors.Is(err, cmd.ErrTestsFailed) {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
