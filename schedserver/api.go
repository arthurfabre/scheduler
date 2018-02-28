// API server implementation
package schedserver

import (
	"context"
	"fmt"
	api "github.com/arthurfabre/scheduler/schedapi"
	"github.com/coreos/etcd/clientv3"
	"google.golang.org/grpc"
	"log"
	"net"
)

type taskServiceServer struct {
	client *clientv3.Client
}

func (s *taskServiceServer) Submit(ctx context.Context, req *api.TaskRequest) (*api.TaskID, error) {
	task := newTask(req)

	err := task.queue(ctx, s.client)
	if err != nil {
		return nil, err
	}

	return task.Id, nil
}

func (s *taskServiceServer) Status(ctx context.Context, id *api.TaskID) (*api.TaskStatus, error) {
	task, err := getTask(ctx, s.client, id)
	if err != nil {
		return nil, err
	}

	return task.Status, nil
}

func (s *taskServiceServer) Cancel(ctx context.Context, id *api.TaskID) (*api.Empty, error) {
	task, err := getTask(ctx, s.client, id)
	if err != nil {
		return nil, err
	}

	err = task.cancel(ctx, s.client)
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

func (s *taskServiceServer) Start(ip string, port uint16) {
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		log.Fatalln("Failed to listen on port %d", port, err)
	}

	grpcServer := grpc.NewServer()
	grpcServer.Serve(lis)
}
