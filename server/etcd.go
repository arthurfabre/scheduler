// ETCD server interface
package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/etcd/embed"
)

// etcdConfig stores the config for etcd
// All fields are required
type etcdConfig struct {
	// name is the name of the node
	name string

	// ip is the IP address etcd will bind
	ip string

	// clientPort is the port that will be bound for clients to talk to us
	clientPort uint16

	// peerPort is the port that will be bound for other nodes to talk to us
	peerPort uint16

	// dataDir is the directory to store data in
	dataDir string

	// nodes is the list of other etcd nodes, in IP:PORT form
	nodes []string

	// newCluster is true if this is a newCluster, false if we're joining an existing one
	newCluster bool

	// timeout is the timeout for etcd to start and the cluster to be joined
	timeout time.Duration

	// context is the context for starting etcd
	ctx context.Context
}

// Get a String URL as slice of url.URLs
func URL(ip string, port uint16) ([]url.URL, error) {
	parsed, err := url.Parse(fmt.Sprintf("http://%s:%d", ip, port))
	if err != nil {
		return nil, err
	}

	return []url.URL{*parsed}, nil
}

// RunEtcd runs an embedded etcd instance. Blocking.
func RunEtcd(c etcdConfig) error {
	cfg := embed.NewConfig()

	cfg.Name = c.name
	cfg.Dir = c.dataDir

	// Other etcd servers
	var err error
	cfg.LPUrls, err = URL(c.ip, c.peerPort)
	if err != nil {
		return err
	}
	// Client
	cfg.LCUrls, err = URL(c.ip, c.clientPort)
	if err != nil {
		return err
	}

	// We only bind the public IP, advertise that
	cfg.APUrls = cfg.LPUrls
	cfg.ACUrls = cfg.LCUrls

	// Cluster includes ourselves and the other nodes
	cfg.InitialCluster = strings.Join(append(c.nodes, cfg.InitialClusterFromName(cfg.Name)), ",")

	if c.newCluster {
		cfg.ClusterState = embed.ClusterStateFlagNew
	} else {
		cfg.ClusterState = embed.ClusterStateFlagExisting
	}

	// We only use v3
	cfg.EnableV2 = false

	e, err := embed.StartEtcd(cfg)
	if err != nil {
		return err
	}
	defer e.Close()

	select {
	case <-e.Server.ReadyNotify():
		log.Printf("ETCD is ready")
	case <-time.After(timeout):
		e.Server.Stop()
		return fmt.Errorf("failed to start etcd server in %v:", timeout)
	case <-c.ctx.Done():
		e.Server.Stop()
	}

	select {
	case <-c.ctx.Done():
		e.Server.Stop()
	case err = <-e.Err():
		return err
	}

	return nil
}
