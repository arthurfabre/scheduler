package main

import (
	"testing"

	"github.com/coreos/etcd/mvcc/mvccpb"
)

// TestTaskID tests task key handling
func TestTaskID(t *testing.T) {
	key := "task/foo"
	id := taskID(key)

	newKey := taskKey(id)

	if newKey != key {
		t.Errorf("taskID(taskID(%s)) != %s, taskID(): %v, taskKey():%s", key, key, id, newKey)
	}
}

// TestStatus

// TestParseTask tests task parsing
func TestParseTask(t *testing.T) {
	key := []byte("task/foo")
	kv := &mvccpb.KeyValue{Key: key, Value: []byte{}}

	_, err := parseTask(kv)
	if err == nil {
		t.Errorf("Expected error, but got %v when parsing empty task", err)
	}

	// TODO - Check all required fields result in error
}

func TestQueue(t *testing.T) {

}
