package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/jessevdk/go-flags"

	"github.com/arthurfabre/scheduler/api"
)

const (
	// Subdirectories of our data dir
	etcdDir      = "etcd"
	containerDir = "container"

	// timeout for starting etcd and the client
	// Needs to be fairly long for static bootstrap to complete
	timeout = 60 * time.Second
)

var opts struct {
	Args struct {
		IP   string `description:"Public IP to bind"`
		Name string `description:"Unique node name"`
	} `positional-args:"true" required:"true"`

	ApiPort uint16 `short:"a" long:"api-port" default:"8080" description:"gRPC task API port"`

	EtcdClientPort uint16 `short:"c" long:"client-port" default:"2379" description:"etcd client port"`

	EtcdPeerPort uint16 `short:"p" long:"peer-port" default:"2380" description:"etcd peer port"`

	DataDir string `short:"d" long:"data-dir" default:"default.etcd" description:"data directory for containers and etcd"`

	Nodes []string `short:"N" long:"node" description:"Other nodes of the cluster to create or join"`

	NewCluster bool `short:"n" long:"new-cluster" description:"Start a new cluster (instead of joining an existing one)"`

	RootFs string `short:"r" long:"root-fs" description:"RootFS used to run tasks in"`
}

// getLog returns the log file location for a given TaskID
func getLog(id *api.TaskID) string {
	return filepath.Join(opts.DataDir, id.Uuid)
}

// start runs a function in a goroutine, writing any errors to e. Non-blocking.
func start(f func() error, e chan<- error) {
	go func() {
		err := f()
		if err != nil {
			e <- err
		}
	}()
}

// client creates an etcd client from the parsed opts
func client() (*clientv3.Client, error) {
	return clientv3.New(clientv3.Config{
		Endpoints:   []string{fmt.Sprintf("http://%s:%d", opts.Args.IP, opts.EtcdClientPort)},
		DialTimeout: timeout,
	})
}

// start starts the server, blocking until an error is encountered
func run() error {
	parser := flags.NewParser(&opts, flags.HelpFlag)
	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			fmt.Println(flagsErr)
			return nil
		}
		return err
	}

	id := nodeID(opts.Args.IP, opts.ApiPort)

	rootCtx, rootCancel := context.WithCancel(context.Background())

	errors := make(chan error)

	start(func() error {
		return RunEtcd(opts.Args.Name, opts.Args.IP, opts.EtcdClientPort, opts.EtcdPeerPort, filepath.Join(opts.DataDir, etcdDir), opts.Nodes, opts.NewCluster, timeout, rootCtx)
	}, errors)

	cli, err := client()
	if err != nil {
		return fmt.Errorf("Error connecting to local etcd: %s", err)
	}

	taskServer := taskServiceServer{cli, id}
	start(func() error {
		return taskServer.Run(opts.Args.IP, opts.ApiPort)
	}, errors)

	runner := Runner{cli, id}
	start(func() error {
		return runner.Run(rootCtx, filepath.Join(opts.DataDir, containerDir), opts.RootFs)
	}, errors)

	err = <-errors
	if err != nil {
		rootCancel()
		return err
	}

	return nil
}

func main() {
	err := run()
	if err != nil {
		log.Fatalln(err)
	}
}
