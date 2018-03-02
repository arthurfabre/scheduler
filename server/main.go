package main

import (
	"context"
	"fmt"
	"github.com/arthurfabre/scheduler/api"
	"github.com/coreos/etcd/clientv3"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
	"path/filepath"
	"time"
)

const (
	// Subdirectories of our data dir
	etcdDir      = "etcd"
	containerDir = "container"
)

var opts struct {
	Args struct {
		Ip   string `description:"Public IP to bind"`
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

func main() {
	if _, err := flags.Parse(&opts); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	id := nodeID(opts.Args.Ip, opts.ApiPort)

	// TODO - use rootCancel()?
	rootCtx, _ := context.WithCancel(context.Background())

	// TODO - Pass rootCtx
	go startEtcd(opts.Args.Name, opts.Args.Ip, opts.EtcdClientPort, opts.EtcdPeerPort, filepath.Join(opts.DataDir, etcdDir), opts.Nodes, opts.NewCluster)

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{fmt.Sprintf("http://%s:%d", opts.Args.Ip, opts.EtcdClientPort)},
		DialTimeout: 2 * time.Second,
	})
	if err != nil {
		log.Fatalln("Error connecting to local etcd:", err)
	}

	taskServer := taskServiceServer{cli, id}
	go taskServer.Start(opts.Args.Ip, opts.ApiPort)

	runner := Runner{cli, id}
	go runner.Start(rootCtx, filepath.Join(opts.DataDir, containerDir), opts.RootFs)

	// TODO - Hacky AF
	for {
		time.Sleep(1 * time.Second)
	}
}
