package main

import (
	"context"
	"log"
	//"github.com/opencontainers/runc/libcontainer" TODO
	"github.com/arthurfabre/scheduler/api"
	"github.com/coreos/etcd/clientv3"
	"time"
)

type Runner struct {
	client *clientv3.Client
	id     *api.NodeID
}

func (r *Runner) Start() {
	newTasks := watchQueuedTasks(context.Background(), r.client)

	for task := range newTasks {
		if err := task.run(context.Background(), r.client, r.id); err != nil {
			// TODO - Differentiate stolen task from other errors
			continue
		}

		log.Println("Running task", task.Id.Uuid)

		// TODO - Actually run it
		go func() {
			runningTask := task
			time.Sleep(10 * time.Second)
			if err := runningTask.complete(context.Background(), r.client, r.id, 0); err != nil {
				log.Println("Error completing task", err)
			}
		}()
	}
}
