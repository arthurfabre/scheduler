package schedserver

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
		Ip string `description:"Public IP to bind"`
	} `positional-args:"true" required:"true"`

	ApiPort uint16 `short:"a" long:"api-port" default:"8080" description:"gRPC task API port"`

	EtcdClientPort uint16 `short:"c" long:"client-port" default:"2379" description:"etcd client port"`

	EtcdPeerPort uint16 `short:"p" long:"peer-port" default:"2380" description:"etcd peer port"`

	//TODO
	// new cluster
	// join cluster
	// cluster members
}

func Main() {
	if _, err := flags.Parse(&opts); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	go startEtcd(opts.Args.Ip, opts.EtcdClientPort, opts.EtcdPeerPort)

	id := nodeID(opts.Args.Ip, opts.ApiPort)

	// TODO - Make etcd() return the EndPoint
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{fmt.Sprintf("http://localhost:%d", opts.EtcdClientPort)},
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
