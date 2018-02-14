package main

import (
    //"fmt"
    //"flag"
    "log"
    "time"
    "net/url"
    //"github.com/opencontainers/runc/libcontainer"
    //"github.com/coreos/etcd/clientv3"
    "github.com/coreos/etcd/embed"
)

// Get a String URL as slice of url.URLs
func URLify(u string) []url.URL {
    // TODO - We probably shouldn't ignore the error?
    parsed, _ := url.Parse(u)
    return []url.URL{*parsed}
}

// Only primitives can be Const.. 
var (
    // Wildcard address for other etcd servers
    PeerURL = URLify("http://0.0.0.0:2380")
    // Localhost only for client
    ClientURL = URLify("http://localhost:2379")

    // TODO - Figure out clustering bootsrap URLs - see https://coreos.com/etcd/docs/latest/v2/configuration.html#clustering-flags
)

func main() {
    // TODO - Move etcd config & startup somewhere else

    cfg := embed.NewConfig()

    // TODO - Is this sane?
    cfg.Dir = "default.etcd"

    cfg.LPUrls = PeerURL
    cfg.LCUrls = ClientURL

    // We only use v3
    cfg.EnableV2 = false

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
