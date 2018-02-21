// API server implementation
package schedserver

import (
	//"fmt"
	//"log"
	//"time"
	"context"
	api "github.com/arthurfabre/scheduler/schedapi"
	"github.com/coreos/etcd/clientv3"
	"google.golang.org/grpc"
)

type taskServiceServer struct {
	client *clientv3.Client
}

func (s *taskServiceServer) Submit(ctx context.Context, req *api.TaskRequest) (*api.TaskID, error) {
	task, err := newTask(s.client, ctx, req)
	if err != nil {
		return nil, err
	}

	err = task.queue(s.client, ctx)
	if err != nil {
		return nil, err
	}

	return task.Id, nil
}

func (s *taskServiceServer) Status(ctx context.Context, id *api.TaskID) (*api.TaskStatus, error) {
	task, err := getTask(s.client, ctx, id)
	if err != nil {
		return nil, err
	}

	return task.Status, nil
}

func (s *taskServiceServer) Cancel(ctx context.Context, id *api.TaskID) (*api.Empty, error) {
	task, err := getTask(s.client, ctx, id)
	if err != nil {
		return nil, err
	}

	err = task.cancel(s.client, ctx)
	if err != nil {
		return nil, err
	}

	return &api.Empty{}, nil
}

func (s *taskServiceServer) Logs(id *api.TaskID, stream api.TaskService_LogsServer) error {
	// TODO
	// Get server from status in /task/UUID
	// If us, read logs
	// If someone else, proxy logs call there
	return nil
}

func startApi() {
	// Prevent unused import for now
	var _ []grpc.ServerOption
}
