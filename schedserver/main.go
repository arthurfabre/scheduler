package schedserver

import (
	//"fmt"
	//"flag"
	//"log"
	//"github.com/opencontainers/runc/libcontainer"
	//"github.com/coreos/etcd/clientv3"
	"github.com/arthurfabre/scheduler/schedapi"
)

var _ = schedapi.TaskID{}

func Main() {
	// TODO - CMD line parsing

	startEtcd()
}
