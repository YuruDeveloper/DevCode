package main

import (
	app "DevCode/src/App"
	"fmt"
)

func main() {
	app, err := app.NewApp()
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return
	}
	app.Run()
}
