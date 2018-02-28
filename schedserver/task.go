package schedserver

import (
	"context"
	"errors"
	"fmt"
	api "github.com/arthurfabre/scheduler/schedapi"
	"github.com/arthurfabre/scheduler/schedserver/pb"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/clientv3util"
	"github.com/golang/protobuf/proto"
	"github.com/satori/go.uuid"
	"log"
	"strings"
	"time"
)

// Format strings for prefixes. A prefix is a key with everything but the last Task UUID component
// See README/#ETCD Key Schema
const (
	taskPrefix        = "task/"
	queuedPrefixFmt   = "task/status/queued/"
	runningPrefixFmt  = "task/status/running/%s/"
	completePrefixFmt = "task/status/complete/%d/"
	canceledPrefixFmt = "task/status/canceled/%d/"
)

// Task handles storing and updating tasks (and their status) in etcd.
// Tasks embeds the pb.Task implementation generated by protoc.
type Task struct {
	*pb.Task

	// Version in etcd when this was last updated / retrieved
	version int64

	// Key in etcd
	key string
}

// watchQueuedTasks returns a Channel of Tasks that have just been queued
// TODO - Potentially expose DELETE events, so we can backoff before stealing tasks
func watchQueuedTasks(ctx context.Context, client *clientv3.Client) <-chan *Task {
	out := make(chan *Task)

	// TODO - Are we cleaning things up properly?
	go func() {
		for resp := range client.Watch(ctx, queuedPrefix(), clientv3.WithPrefix(), clientv3.WithFilterDelete()) {
			for _, ev := range resp.Events {
				task, err := getTask(ctx, client, keyID(string(ev.Kv.Key)))
				if err != nil {
					log.Println("Error retrieving task", ev.Kv.Key)
					continue
				}

				out <- task
			}
		}
	}()

	return out
}

// listDoneTasks returns a list of Tasks that were done (completed or canceled) at least age seconds ago.
func listDoneTasks(ctx context.Context, client *clientv3.Client, age int64) ([]*Task, error) {
	// Get everything from epoch 0 to (Now - age)
	end := time.Now().Unix() - age

	completed, err := listTasks(ctx, client, completePrefix(0), clientv3.WithRange(completePrefix(end)))
	if err != nil {
		return nil, err
	}

	canceled, err := listTasks(ctx, client, canceledPrefix(0), clientv3.WithRange(canceledPrefix(end)))
	if err != nil {
		return nil, err
	}

	return append(completed, canceled...), nil
}

// listNodeTasks returns a list of Tasks that are being run by nodeId.
func listNodeTasks(ctx context.Context, client *clientv3.Client, nodeId *api.NodeID) ([]*Task, error) {
	return listTasks(ctx, client, runningPrefix(nodeId), clientv3.WithPrefix())
}

// listTasks returns a list of Tasks using etcd GET(key, opts...). Intended to be used with status keys.
func listTasks(ctx context.Context, client *clientv3.Client, key string, opts ...clientv3.OpOption) ([]*Task, error) {
	resp, err := client.Get(ctx, key, opts...)
	if err != nil {
		return nil, err
	}

	tasks := make([]*Task, 0, resp.Count)

	for _, t := range resp.Kvs {
		task, err := getTask(ctx, client, keyID(string(t.Key)))
		if err != nil {
			log.Println("Error retrieving task", t.Key)
			continue
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// newTask constructs a Task from a TaskRequest, assigining it a UUID.
// Nothing is submitted to etcd.
func newTask(req *api.TaskRequest) *Task {
	id := &api.TaskID{uuid.NewV4().String()}

	return &Task{key: taskKey(id), Task: &pb.Task{Request: req, Id: id}}
}

// getTask retrieves a Task from etcd.
func getTask(ctx context.Context, client *clientv3.Client, id *api.TaskID) (*Task, error) {
	key := taskKey(id)

	resp, err := client.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	// We're not searching for a range or prefix
	if resp.Count != 1 {
		return nil, fmt.Errorf("Too many matching keys, found %d", resp.Count)
	}

	task := &Task{version: resp.Kvs[0].Version, key: key, Task: &pb.Task{}}
	if err := proto.Unmarshal([]byte(resp.Kvs[0].Value), task.Task); err != nil {
		return nil, err
	}

	// Ensure what we requested and what we got back match up
	if task.Id.Uuid != id.Uuid {
		return nil, fmt.Errorf("Requested task %s, received %s", id.Uuid, task.Id.Uuid)
	}

	return task, nil
}

// taskKey returns the etcd key for a TaskID
func taskKey(id *api.TaskID) string {
	return idKey(taskPrefix, id)
}

// queuedPrefix returns the status key prefix for queued tasks
func queuedPrefix() string {
	return queuedPrefixFmt
}

// runningPrefix returns the status key prefix for tasks running on id
func runningPrefix(id *api.NodeID) string {
	return fmt.Sprintf(runningPrefixFmt, id.Uuid)
}

// completePrefix returns the status key prefix for tasks completed at age
func completePrefix(age int64) string {
	return fmt.Sprintf(completePrefixFmt, age)
}

// canceledPrefix returns the status key prefix for tasks canceledat age
func canceledPrefix(age int64) string {
	return fmt.Sprintf(canceledPrefixFmt, age)
}

// statusKey returns the etcd status key of a Task for a given TaskStatus
func (t *Task) statusKey(status *api.TaskStatus) string {
	var prefix string

	switch status.Status.(type) {
	case *api.TaskStatus_Queued_:
		prefix = queuedPrefix()
	case *api.TaskStatus_Running_:
		prefix = runningPrefix(status.GetRunning().NodeId)
	case *api.TaskStatus_Complete_:
		prefix = completePrefix(status.GetComplete().Epoch)
	case *api.TaskStatus_Canceled_:
		prefix = canceledPrefix(status.GetCanceled().Epoch)
	default:
		// TODO - Is this wise?
		panic("Unexpected Task status")
	}

	return idKey(prefix, t.Id)
}

// keyID converts a status / task key, to a TaskID
func keyID(key string) *api.TaskID {
	s := strings.Split(key, "/")
	return &api.TaskID{Uuid: s[len(s)-1]}
}

// idKey converts a key prefix to full status / task key
func idKey(prefix string, key *api.TaskID) string {
	return fmt.Sprintf(prefix+"%s", key.Uuid)
}

// setStatus Updates the status of a Task, and updates the Task and its status key in etcd
func (t *Task) setStatus(ctx context.Context, client *clientv3.Client, newStatus *api.TaskStatus) (err error) {
	// Preserve old status to know which old key to delete
	oldStatus := t.Status
	t.Status = newStatus

	// If there's an error, ensure we set the oldStatus back
	defer func() {
		if err != nil {
			t.Status = oldStatus
		}
	}()

	data, err := proto.Marshal(t.Task)
	if err != nil {
		return
	}

	kvc := clientv3.NewKV(client)

	// We always need to update the task and its status key
	thens := []clientv3.Op{
		clientv3.OpPut(t.key, string(data)),
		clientv3.OpPut(t.statusKey(newStatus), ""),
	}

	// If a previous status was set, cleanup its key
	if oldStatus != nil {
		thens = append(thens, clientv3.OpDelete(t.statusKey(oldStatus)))
	}

	var ifCheck clientv3.Cmp
	if t.version > 0 {
		// If the Task was already stored, ensure it hasn't changed
		ifCheck = clientv3.Compare(clientv3.Version(t.key), "=", t.version)
	} else {
		// If it was never stored, ensure no one else has stolen that key
		ifCheck = clientv3util.KeyMissing(t.key)
	}

	resp, err := kvc.Txn(ctx).If(ifCheck).Then(thens...).Commit()

	if err != nil {
		return
	}

	if !resp.Succeeded {
		err = errors.New("Unexpected concurrent Task modification")
		return
	}

	// Feels hacky, but PutResponse doesn't include the new version
	t.version++

	return nil
}

// queue marks the Task as "queued" in etcd.
func (t *Task) queue(ctx context.Context, client *clientv3.Client) error {
	return t.setStatus(ctx, client, &api.TaskStatus{&api.TaskStatus_Queued_{&api.TaskStatus_Queued{}}})
}

// run marks the Task as "running" on nodeID in etcd.
func (t *Task) run(ctx context.Context, client *clientv3.Client, nodeID *api.NodeID) error {
	return t.setStatus(ctx, client, &api.TaskStatus{&api.TaskStatus_Running_{&api.TaskStatus_Running{nodeID}}})
}

// complete marks the Task as "complete" on nodeID, with exitCode, as of now, in etcd.
func (t *Task) complete(ctx context.Context, client *clientv3.Client, nodeID *api.NodeID, exitCode int) error {
	return t.setStatus(ctx, client, &api.TaskStatus{&api.TaskStatus_Complete_{&api.TaskStatus_Complete{nodeID, int32(exitCode), time.Now().Unix()}}})
}

// cancel marks the Task as "canceled" as of now, in etcd.
func (t *Task) cancel(ctx context.Context, client *clientv3.Client) error {
	return t.setStatus(ctx, client, &api.TaskStatus{&api.TaskStatus_Canceled_{&api.TaskStatus_Canceled{time.Now().Unix()}}})
}
