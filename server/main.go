package main

import (
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
	"time"
)

var opts struct {
	Args struct {
		Ip   string `description:"Public IP to bind"`
		Name string `description:"Unique node name"`
	} `positional-args:"true" required:"true"`

	ApiPort uint16 `short:"a" long:"api-port" default:"8080" description:"gRPC task API port"`

	EtcdClientPort uint16 `short:"c" long:"client-port" default:"2379" description:"etcd client port"`

	EtcdPeerPort uint16 `short:"p" long:"peer-port" default:"2380" description:"etcd peer port"`

	EtcdDataDir string `short:"d" long:"data-dir" default:"default.etcd" description:"etcd data directory"`

	Nodes []string `short:"N" long:"node" description:"Other nodes of the cluster to create or join"`

	NewCluster bool `short:"n" long:"new-cluster" description:"Start a new cluster (instead of joining an existing one)"`
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

	go startEtcd(opts.Args.Name, opts.Args.Ip, opts.EtcdClientPort, opts.EtcdPeerPort, opts.EtcdDataDir, opts.Nodes, opts.NewCluster)

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{fmt.Sprintf("http://%s:%d", opts.Args.Ip, opts.EtcdClientPort)},
		DialTimeout: 2 * time.Second,
	})
	if err != nil {
		log.Fatalln("Error connecting to local etcd:", err)
	}

	taskServer := taskServiceServer{cli}
	go taskServer.Start(opts.Args.Ip, opts.ApiPort)

	runner := Runner{cli, id}
	go runner.Start()

	// TODO - Hacky AF
	for {
		time.Sleep(1 * time.Second)
	}
}
