package main

import (
	"github.com/arthurfabre/scheduler/api"
	"github.com/jessevdk/go-flags"
	"google.golang.org/grpc"
	"log"
	"os"
)

// Hacky globals for cli parsing...
var opts struct {
	Node string `short:"N" long:"node" default:"127.0.0.1:8080" description:"Node endpoint to use"`
}
var parser = flags.NewParser(&opts, flags.Default)

func main() {
	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}
}

// getClient re
func getClient() api.TaskServiceClient {
	conn, err := grpc.Dial(opts.Node, grpc.WithInsecure())
	if err != nil {
		log.Fatalln("Error connecting to node", err)
	}

	return api.NewTaskServiceClient(conn)
}
