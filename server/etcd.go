// ETCD server interface
package main

import (
	"fmt"
	"github.com/coreos/etcd/embed"
	"log"
	"net/url"
	"strings"
	"time"
)

// Get a String URL as slice of url.URLs
func URL(ip string, port uint16) []url.URL {
	// TODO - We probably shouldn't ignore the error?
	parsed, _ := url.Parse(fmt.Sprintf("http://%s:%d", ip, port))
	return []url.URL{*parsed}
}

func startEtcd(name string, ip string, clientPort uint16, peerPort uint16, dataDir string, nodes []string, newCluster bool) {
	cfg := embed.NewConfig()

	cfg.Name = name
	cfg.Dir = dataDir

	// Other etcd servers
	cfg.LPUrls = URL(ip, peerPort)
	// Client
	cfg.LCUrls = URL(ip, clientPort)

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

	// TODO - This is example code, check error handling
	e, err := embed.StartEtcd(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer e.Close()
	select {
	case <-e.Server.ReadyNotify():
		log.Printf("Server is ready!")
	case <-time.After(60 * time.Second):
		e.Server.Stop() // trigger a shutdown
		log.Printf("Server took too long to start!")
	}

	log.Fatal(<-e.Err())
}
