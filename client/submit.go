package main

import (
	"context"
	"github.com/arthurfabre/scheduler/api"
	"log"
)

type submitCommand struct {
	Args struct {
		Command string   `description:"Command to run"`
		Args    []string `description:"Arguments to pass to Command"`
	} `positional-args:"true"`

	// TODO - RAM, CPU...
}

func init() {
	parser.AddCommand("run", "Queue a task to be run", "", &submitCommand{})
}

func (s *submitCommand) Execute(args []string) error {
	client := getClient()

	id, err := client.Submit(context.Background(), &api.TaskRequest{s.Args.Command, s.Args.Args})
	if err != nil {
		log.Fatalln("Error queuing task", err)
	}

	log.Println("Task submitted as", id.Uuid)

	return nil
}
