package main

// Wrapper to ensure go build / go get work with the root of this repo

import "github.com/arthurfabre/scheduler/schedserver"

func main() {
	schedserver.Main()
}
