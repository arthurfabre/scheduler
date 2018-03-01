package main

import (
	"github.com/arthurfabre/scheduler/api"
	"github.com/satori/go.uuid"
)

// nodeID returns a NodeID with new random UUID
func nodeID(ip string, apiPort uint16) *api.NodeID {
	return &api.NodeID{uuid.NewV4().String(), ip, int32(apiPort)}
}

// TODO
/*func registerNode(client *clientv3.Client, id *api.NodeID) (*api.NodeID, error) {
	id := nodeID

	resp, err := client.Grant(context.Background(), 30)
	if err != nil {
		log.Fatal(err)
	}

	_, err = cli.Put(context.TODO(), "foo", "bar", clientv3.WithLease(resp.ID))
	if err != nil {
		log.Fatal(err)
	}

	// the key 'foo' will be kept forever
	ch, kaerr := cli.KeepAlive(context.TODO(), resp.ID)
	if kaerr != nil {
		log.Fatal(kaerr)
	}
}

func watchDeadNodes(client *clientv3.Client, ctx context.Context) <-chan *api.NodeID {

}*/
