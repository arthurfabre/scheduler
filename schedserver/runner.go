package schedserver

import (
	"context"
	"log"
	//"github.com/opencontainers/runc/libcontainer" TODO
	api "github.com/arthurfabre/scheduler/schedapi"
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
			time.Sleep(10 * time.Second)
			task.complete(context.Background(), r.client, r.id, 0)
		}()
	}
}