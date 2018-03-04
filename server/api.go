// API server implementation
package main

import (
	"context"
	"fmt"
	"io"
	"net"

	"github.com/arthurfabre/scheduler/api"
	"github.com/coreos/etcd/clientv3"
	"github.com/hpcloud/tail"
	"google.golang.org/grpc"
)

type taskServiceServer struct {
	client *clientv3.Client
	id     *api.NodeID
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

	// Check the task's status allows being canceled
	switch task.Status.Status.(type) {
	case *api.TaskStatus_Queued_:
		// OK
	case *api.TaskStatus_Running_:
		// OK
	case *api.TaskStatus_Complete_:
		return nil, fmt.Errorf("Task %s is already complete", id.Uuid)
	case *api.TaskStatus_Canceled_:
		return nil, fmt.Errorf("Task %s is already canceled", id.Uuid)
	default:
		return nil, fmt.Errorf("Task %s unknown status", id.Uuid)
	}

	err = task.cancel(ctx, s.client)
	if err != nil {
		return nil, err
	}

	return &api.Empty{}, nil
}

func (s *taskServiceServer) Logs(id *api.TaskID, stream api.TaskService_LogsServer) error {
	task, err := getTask(stream.Context(), s.client, id)
	if err != nil {
		return err
	}

	// Id of the node running the task
	var nodeId *api.NodeID
	// True if the task is done
	var isDone bool

	switch task.Status.Status.(type) {
	case *api.TaskStatus_Queued_:
		return fmt.Errorf("Task %s is queued", id.Uuid)
	case *api.TaskStatus_Running_:
		nodeId = task.Status.GetRunning().NodeId
		isDone = false
	case *api.TaskStatus_Complete_:
		nodeId = task.Status.GetComplete().NodeId
		isDone = true
	case *api.TaskStatus_Canceled_:
		return fmt.Errorf("Task %s is canceled", id.Uuid)
	default:
		return fmt.Errorf("Task %s unknown status", id.Uuid)
	}

	// We're not running / handling the task, proxy to the node that is
	if nodeId.Uuid != s.id.Uuid {
		conn, err := grpc.Dial(fmt.Sprintf("%s:%d", nodeId.Ip, nodeId.Port), grpc.WithInsecure())
		if err != nil {
			return err
		}
		defer conn.Close()

		client := api.NewTaskServiceClient(conn)

		logs, err := client.Logs(stream.Context(), id)
		if err != nil {
			return err
		}

		for {
			log, err := logs.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}

			err = stream.Send(log)
			if err != nil {
				return err
			}
		}
		return nil
	}

	// tail -f the log file if the task is not done
	// TODO - If the task finished in the meantime, we won't know
	logFile, err := tail.TailFile(getLog(id), tail.Config{Follow: !isDone})
	if err != nil {
		return err
	}

	for line := range logFile.Lines {
		err = stream.Send(&api.Log{[]string{line.Text}})
		if err != nil {
			return err
		}
	}

	return nil
}

// Run runs the gRPC server for the API. Blocking.
func (s *taskServiceServer) Run(ip string, port uint16) error {
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	api.RegisterTaskServiceServer(grpcServer, s)
	err = grpcServer.Serve(lis)
	if err != nil {
		return err
	}

	return nil
}
