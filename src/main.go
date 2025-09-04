package main

import (
	app "DevCode/src/App"
	"fmt"
	"os"
)

func main() {
	app, err := app.NewApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing application: %v\n", err)
		os.Exit(1)
	}
	app.Run()
}
