// API server implementation
package schedserver

import (
	//"fmt"
	//"log"
	//"time"
	"context"
	api "github.com/arthurfabre/scheduler/schedapi"
	"google.golang.org/grpc"
)

type taskServiceServer struct {
}

func (s *taskServiceServer) Submit(ctx context.Context, req *api.TaskRequest) (*api.TaskID, error) {
	// TODO
	// Generate UUID
	// Write Task proto to /task/UUID
	// Write /status/queued/UUID
	return nil, nil
}

func (s *taskServiceServer) Status(ctx context.Context, id *api.TaskID) (*api.TaskStatus, error) {
	// TODO
	// Get /task/UUID
	return nil, nil
}

func (s *taskServiceServer) Cancel(ctx context.Context, id *api.TaskID) (*api.Empty, error) {
	// TODO
	// Update status in /task/UUID
	// Running server will handle queue shenanigans
	return nil, nil
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
