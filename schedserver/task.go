package schedserver

import (
	"context"
	api "github.com/arthurfabre/scheduler/schedapi"
	"github.com/coreos/etcd/clientv3"
	"github.com/golang/protobuf/proto"
	"log"
)

const (
	keySep    = "/"
	taskPrefix = "task" + keySep
	statusPrefix = taskPrefix + "status" + keySep
)

// Utility methods for interfacing ProtocolBuffer Tasks and etcd.
// Handles keyspace, and provides transactions for updating the status of Tasks.

// NewTask constructs a Task from a TaskRequest, assigining it a UUID.
// Status defaults to Queued
func newTask(client *clientv3.Client, ctx context.Context, req *api.TaskRequest) (*Task, error) {
	// TODO - make UUID, queue()
	return nil, nil
}

func getTask(client *clientv3.Client, ctx context.Context, id *api.TaskID) (*Task, error) {
	resp, err := client.Get(ctx, taskID2Key(id))
	if err != nil {
		return nil, err
	}

	// We're not searching for a range or prefix
	if resp.Count != 1 {
		log.Fatalln("Too many matching keys, found:", resp.Count)
	}

	task := &Task{}
	if err := proto.Unmarshal([]byte(resp.Kvs[0].Value), task); err != nil {
		return nil, err
	}

	return task, nil
}

func taskID2Key(id *api.TaskID) string {
	return taskPrefix + id.Uuid
}

func taskStatus2Key(status *api.TaskStatus) string {
	// TODO - How to get status name?
	return nil
}

func (t *Task) setStatus(client *clientv3.Client, ctx context.Context, s *api.TaskStatus) error {
	// TODO - encode, write to etcd using TXN a
	return nil
}

func (t *Task) queue(client *clientv3.Client, ctx context.Context) error {
	return t.setStatus(client, ctx, &api.TaskStatus{&api.TaskStatus_Queued_{}})
}

func (t *Task) cancel(client *clientv3.Client, ctx context.Context) error {
	return t.setStatus(client, ctx, &api.TaskStatus{&api.TaskStatus_Cancelled_{}})
}

func (t *Task) run(client *clientv3.Client, ctx context.Context, nodeID *api.NodeID) error {
	return t.setStatus(client, ctx, &api.TaskStatus{&api.TaskStatus_Running_{&api.TaskStatus_Running{nodeID}}})
}

// Complete updates the status of the Task to complete, with the given exit code.
func (t *Task) complete(client *clientv3.Client, ctx context.Context, nodeID *api.NodeID, exitCode int) error {
	return t.setStatus(client, ctx, &api.TaskStatus{&api.TaskStatus_Complete_{&api.TaskStatus_Complete{nodeID, int32(exitCode)}}})
}
