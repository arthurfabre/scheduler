package main

import (
	"context"
	"log"

	"github.com/arthurfabre/scheduler/api"
)

type statusCommand struct {
	Args struct {
		Id string `description:"UUID of the task to check the status of" required:"true"`
	} `positional-args:"true"`
}

func init() {
	parser.AddCommand("status", "Get the status of a task", "", &statusCommand{})
}

func (s *statusCommand) Execute(args []string) error {
	client := getClient()

	status, err := client.Status(context.Background(), &api.TaskID{s.Args.Id})
	if err != nil {
		log.Fatalln("Error checking status", err)
	}

	log.Println("Status:", status)

	return nil
}
