package main

import (
	"context"
	"io"
	"log"

	"github.com/arthurfabre/scheduler/api"
)

type logsCommand struct {
	Args struct {
		Id string `description:"UUID of the task to display the logs of" required:"true"`
	} `positional-args:"true"`
}

func init() {
	parser.AddCommand("logs", "Display the output of a task", "", &logsCommand{})
}

func (s *logsCommand) Execute(args []string) error {
	client := getClient()

	logStream, err := client.Logs(context.Background(), &api.TaskID{s.Args.Id})
	if err != nil {
		log.Fatalln("Error getting logs", err)
	}

	for {
		logEntry, err := logStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalln("Error retrieving log", err)
		}

		for _, line := range logEntry.Line {
			log.Println(line)
		}
	}

	return nil
}
