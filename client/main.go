package main

import (
	"context"
	"github.com/arthurfabre/scheduler/api"
	"github.com/jessevdk/go-flags"
	"google.golang.org/grpc"
	"log"
	"os"
)

// TODO - seperate subcommand for every operation (Submit, Cancel, Status, Logs)
var opts struct {
	Args struct {
		Command string   `description:"Public IP to bind" required:"true"`
		Args    []string `description:"Unique node name"`
	} `positional-args:"true"`

	Node string `short:"N" long:"node" default:"127.0.0.1:8080" description:"Node to submit job with"`

	// TODO - RAM, CPU...
}

func main() {
	if _, err := flags.Parse(&opts); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	conn, err := grpc.Dial(opts.Node, grpc.WithInsecure())
	if err != nil {
		log.Fatalln("Error connecting to node", err)
	}
	defer conn.Close()

	client := api.NewTaskServiceClient(conn)

	id, err := client.Submit(context.Background(), &api.TaskRequest{opts.Args.Command, opts.Args.Args})
	if err != nil {
		log.Fatalln("Error queuing task", err)
	}

	log.Println("Task submitted as", id.Uuid)
}
