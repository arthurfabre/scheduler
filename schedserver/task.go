package schedserver

import (
	"context"
	api "github.com/arthurfabre/scheduler/schedapi"
	"github.com/coreos/etcd/clientv3"
	"github.com/golang/protobuf/proto"
	"log"
	"strconv"
	"time"
)

const (
	keySep       = "/"
	taskPrefix   = "task" + keySep
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
	resp, err := client.Get(ctx, taskKey(id))
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

func taskKey(id *api.TaskID) string {
	return taskPrefix + id.Uuid
}

func (t *Task) statusKey(status *api.TaskStatus) string {
	key := statusPrefix

	// See README/#ETCD Key Schema
	switch status.Status.(type) {
	case *api.TaskStatus_Queued_:
		key += "queued"
	case *api.TaskStatus_Running_:
		key += "running" + keySep + status.GetRunning().NodeId.Uuid
	case *api.TaskStatus_Complete_:
		key += "complete" + keySep + strconv.FormatInt(status.GetComplete().Epoch, 10)
	case *api.TaskStatus_Canceled_:
		key += "canceled" + keySep + strconv.FormatInt(status.GetCanceled().Epoch, 10)
	}

	return key + keySep + t.Id.Uuid
}

func (t *Task) setStatus(client *clientv3.Client, ctx context.Context, s *api.TaskStatus) error {
	//kvc := clientv3.NewKV(client)

	// TODO - Need revision of Task to implement CAS
	//_, err := kvc.Txn(ctx).
	//	If(clientv3.  )

	// TODO - encode, write to etcd using TXN a
	return nil
}

func (t *Task) queue(client *clientv3.Client, ctx context.Context) error {
	return t.setStatus(client, ctx, &api.TaskStatus{&api.TaskStatus_Queued_{&api.TaskStatus_Queued{}}})
}

func (t *Task) run(client *clientv3.Client, ctx context.Context, nodeID *api.NodeID) error {
	return t.setStatus(client, ctx, &api.TaskStatus{&api.TaskStatus_Running_{&api.TaskStatus_Running{nodeID}}})
}

// Complete updates the status of the Task to complete, with the given exit code.
func (t *Task) complete(client *clientv3.Client, ctx context.Context, nodeID *api.NodeID, exitCode int) error {
	return t.setStatus(client, ctx, &api.TaskStatus{&api.TaskStatus_Complete_{&api.TaskStatus_Complete{nodeID, int32(exitCode), time.Now().Unix()}}})
}

func (t *Task) cancel(client *clientv3.Client, ctx context.Context) error {
	return t.setStatus(client, ctx, &api.TaskStatus{&api.TaskStatus_Canceled_{&api.TaskStatus_Canceled{time.Now().Unix()}}})
}
