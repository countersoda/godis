package main

import "github.com/countersoda/godis/app"

func main() {
	app.NewGodis("localhost:6379")
}
