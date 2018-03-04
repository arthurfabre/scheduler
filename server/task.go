package main

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/clientv3util"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/golang/protobuf/proto"
	"github.com/satori/go.uuid"

	"github.com/arthurfabre/scheduler/api"
	"github.com/arthurfabre/scheduler/server/pb"
)

// Format strings for prefixes. A prefix is a key with everything but the last Task UUID component
// See README/#ETCD Key Schema
const (
	taskPrefix        = "task/"
	queuedPrefixFmt   = "task/status/queued/"
	runningPrefixFmt  = "task/status/running/%s/"
	completePrefixFmt = "task/status/complete/%d/"
	canceledPrefixFmt = "task/status/canceled/%d/"
	failedPrefixFmt   = "task/status/failed/"
)

var (
	ConcurrentTaskModErr = errors.New("Concurrent task modification")
)

// Task handles storing and updating tasks (and their status) in etcd.
// Tasks embeds the pb.Task implementation generated by protoc.
type Task struct {
	*pb.Task

	// Version in etcd when this was last updated / retrieved
	version int64

	// modRevision is the Etcd revision when this was last modified, as of when this was last updated / retrieved
	modRevision int64

	// Key in etcd
	key string
}

// TaskEvent is a union / sum type of possible events
type TaskEvent interface {
	isTaskEvent()
}

// TaskUpdate means a task was modified in etcd
type TaskUpdate struct {
	// The updated task
	task *Task
}

func (t TaskUpdate) isTaskEvent() {}

// TaskDelete means a task was deleted in etcd
type TaskDelete struct {
	id *api.TaskID
}

func (t TaskDelete) isTaskEvent() {}

// TaskError means a task was updated / deleted, but an error occured
type TaskError struct {
	err error
	id  *api.TaskID // may be nil
}

func (t TaskError) isTaskEvent() {}

// watchQueuedTasks returns a Channel of TaskEvents for Tasks that have just been queued
func watchQueuedTasks(ctx context.Context, client *clientv3.Client) <-chan TaskEvent {
	return watchTasks(ctx, client, queuedPrefix(), clientv3.WithPrefix())
}

// watch returns a Channel of TaskEvent updates to this Task since this task was last updated / retrieved
func (t *Task) watch(ctx context.Context, client *clientv3.Client) <-chan TaskEvent {
	// Revision + 1 so we don't get the last modification we're aware of, but the next
	return watchTasks(ctx, client, t.key, clientv3.WithRev(t.modRevision+1))
}

// watchTasks returns a Channel of TaskEvent using etcd WATCH(key, opts...)
func watchTasks(ctx context.Context, client *clientv3.Client, key string, opts ...clientv3.OpOption) <-chan TaskEvent {
	out := make(chan TaskEvent)

	go func() {
		defer close(out)

		for resp := range client.Watch(ctx, key, opts...) {
			if resp.Err() != nil {
				out <- TaskError{resp.Err(), nil}
				return
			}

			for _, ev := range resp.Events {
				// TODO - we should be able to just `task, err := parseTask(ev.Kv)`, but ev.Kv.Value is nil..
				task, err := getTask(ctx, client, taskID(string(ev.Kv.Key)))

				if err != nil {
					out <- TaskError{err, taskID(string(ev.Kv.Key))}
					continue
				}

				switch ev.Type {
				case mvccpb.DELETE:
					out <- TaskDelete{task.Id}

				case mvccpb.PUT:
					out <- TaskUpdate{task}
				}
			}
		}
	}()

	return out
}

// listDoneTasks returns a list of TaskEvents (no TaskDelete) that were done (completed or canceled) at least age seconds ago.
func listDoneTasks(ctx context.Context, client *clientv3.Client, age int64) ([]TaskEvent, error) {
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

// listNodeTasks returns a list of TaskEvents (no TaskDelete) that are being run by nodeId.
func listNodeTasks(ctx context.Context, client *clientv3.Client, nodeId *api.NodeID) ([]TaskEvent, error) {
	return listTasks(ctx, client, runningPrefix(nodeId), clientv3.WithPrefix())
}

// listTasks returns a list of TaskEvents (no TaskDelete) using etcd GET(key, opts...). Intended to be used with status keys.
func listTasks(ctx context.Context, client *clientv3.Client, key string, opts ...clientv3.OpOption) ([]TaskEvent, error) {
	resp, err := client.Get(ctx, key, opts...)
	if err != nil {
		return nil, err
	}

	tasks := make([]TaskEvent, 0, resp.Count)

	for _, t := range resp.Kvs {
		task, err := parseTask(t)

		if err != nil {
			tasks = append(tasks, TaskError{err, taskID(string(t.Key))})
			continue
		}

		tasks = append(tasks, TaskUpdate{task})
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
	if len(resp.Kvs) != 1 {
		return nil, fmt.Errorf("Expected a single key match, got %d", resp.Count)
	}

	return parseTask(resp.Kvs[0])
}

// parseTask unmarshals / parses a Task from an etcd KV
func parseTask(kv *mvccpb.KeyValue) (*Task, error) {
	key := string(kv.Key)
	task := &Task{version: kv.Version, modRevision: kv.ModRevision, key: key, Task: &pb.Task{}}

	if err := proto.Unmarshal([]byte(kv.Value), task.Task); err != nil {
		return nil, err
	}

	// Ensure the task matches its key
	if task.Id.Uuid != taskID(key).Uuid {
		return nil, fmt.Errorf("Key mismatch key: %s, proto: %s", key, task.Id.Uuid)
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

// canceledPrefix returns the status key prefix for tasks canceled at age
func canceledPrefix(age int64) string {
	return fmt.Sprintf(canceledPrefixFmt, age)
}

// failedPrefix returns the status key prefix for failed tasks
func failedPrefix() string {
	return failedPrefixFmt
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
	case *api.TaskStatus_Failed_:
		prefix = failedPrefix()
	default:
		// TODO - Is this wise?
		panic("Unexpected Task status")
	}

	return idKey(prefix, t.Id)
}

// taskID converts a status / task key, to a TaskID
func taskID(key string) *api.TaskID {
	s := strings.Split(key, "/")
	return &api.TaskID{Uuid: s[len(s)-1]}
}

// idKey converts a key prefix to full status / task key
func idKey(prefix string, key *api.TaskID) string {
	return prefix + key.Uuid
}

// setStatus Updates the status of a Task, and updates the Task and its status key in etcd
// err is a ConcurrentTaskModErr IFF the task was modified before we could set the status
func (t *Task) setStatus(ctx context.Context, client *clientv3.Client, newStatus *api.TaskStatus) (err error) {
	// Disallow changing to the same status
	if t.Status != nil && reflect.TypeOf(t.Status.Status) == reflect.TypeOf(newStatus.Status) {
		return fmt.Errorf("Task already has status %T", t.Status.Status)
	}

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
		// Don't reorder without fixing how to get modRevision!
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
		err = ConcurrentTaskModErr
		return
	}

	// Feels hacky, but PutResponse doesn't include the new version
	t.version++

	// TXN count as one revision, doesn't matter which of resp.Responses() we look at.
	t.modRevision = resp.Header.Revision

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

// queue marks the Task as "error" with msg, in etcd.
func (t *Task) fail(ctx context.Context, client *clientv3.Client, err error) error {
	return t.setStatus(ctx, client, &api.TaskStatus{&api.TaskStatus_Failed_{&api.TaskStatus_Failed{err.Error()}}})
}
