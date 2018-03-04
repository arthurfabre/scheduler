package main

import (
	"context"
	"log"

	"github.com/arthurfabre/scheduler/api"
)

type cancelCommand struct {
	Args struct {
		Id string `description:"UUID of the task to cancel" required:"true"`
	} `positional-args:"true"`
}

func init() {
	parser.AddCommand("cancel", "Cancel a task", "", &cancelCommand{})
}

func (s *cancelCommand) Execute(args []string) error {
	client := getClient()

	_, err := client.Cancel(context.Background(), &api.TaskID{s.Args.Id})
	if err != nil {
		log.Fatalln("Error canceling task", err)
	}

	log.Println("Task canceled")

	return nil
}
