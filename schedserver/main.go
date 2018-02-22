package schedserver

import (
	//"flag"
	"fmt"
	"log"
	//"github.com/opencontainers/runc/libcontainer"
	"github.com/coreos/etcd/clientv3"
	"time"
)

const (
	// apiPort is the port for the gRPC task submission API
	apiPort = 8080
	// etcdClientPort is the port for the etcd KV API
	etcdClientPort = 2379
	// etcdClusterPort is the port for inter-cluster etcd comms
	etcdClusterPort = 2380
)

func Main() {
	// TODO - CMD line parsing

	go startEtcd()

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{fmt.Sprintf("http://localhost:%d", etcdClientPort)},
		DialTimeout: 2 * time.Second,
	})
	if err != nil {
		log.Fatalln("Error connecting to local etcd:", err)
	}

	taskServer := taskServiceServer{cli}
	go taskServer.Start(apiPort)

	// TODO - Hacky AF
	for {
		time.Sleep(1 * time.Second)
	}
	// startMaintainer()
	// startRunner()
	// runner watches /status/queued for new tasks
	// runner watches tasks we've completed until they're deleted by the maintainer
}
