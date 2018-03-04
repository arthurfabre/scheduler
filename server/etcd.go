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

// Get a String URL as slice of url.URLs
func URL(ip string, port uint16) ([]url.URL, error) {
	parsed, err := url.Parse(fmt.Sprintf("http://%s:%d", ip, port))
	if err != nil {
		return nil, err
	}

	return []url.URL{*parsed}, nil
}

// RunEtcd runs an embedded etcd instance. Blocking.
func RunEtcd(name string, ip string, clientPort uint16, peerPort uint16, dataDir string, nodes []string, newCluster bool, timeout time.Duration, ctx context.Context) error {
	cfg := embed.NewConfig()

	cfg.Name = name
	cfg.Dir = dataDir

	// Other etcd servers
	var err error
	cfg.LPUrls, err = URL(ip, peerPort)
	if err != nil {
		return err
	}
	// Client
	cfg.LCUrls, err = URL(ip, clientPort)
	if err != nil {
		return err
	}

	// We only bind the public IP, advertise that
	cfg.APUrls = cfg.LPUrls
	cfg.ACUrls = cfg.LCUrls

	// Cluster includes ourselves and the other nodes
	cfg.InitialCluster = strings.Join(append(nodes, cfg.InitialClusterFromName(cfg.Name)), ",")

	if newCluster {
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
		log.Printf("Server is ready!")
	case <-time.After(timeout):
		e.Server.Stop()
		return fmt.Errorf("Failed to start etcd server in:", timeout)
	case <-ctx.Done():
		e.Server.Stop()
	}

	select {
	case <-ctx.Done():
		e.Server.Stop()
	case err = <-e.Err():
		return err
	}

	return nil
}
