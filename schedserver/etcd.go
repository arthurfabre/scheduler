// ETCD server interface
package schedserver

import (
	"fmt"
	"github.com/coreos/etcd/embed"
	"log"
	"net/url"
	"time"
)

// Get a String URL as slice of url.URLs
func newURL(u string) []url.URL {
	// TODO - We probably shouldn't ignore the error?
	parsed, _ := url.Parse(u)
	return []url.URL{*parsed}
}

func startEtcd() {
	cfg := embed.NewConfig()

	// TODO - Is this sane?
	cfg.Dir = "default.etcd"

	// Wildcard address for other etcd servers
	cfg.LPUrls = newURL("http://0.0.0.0:2380")
	// Localhost only for client
	cfg.LCUrls = newURL("http://localhost:2379")
	// TODO - Figure out clustering bootsrap URLs - see https://coreos.com/etcd/docs/latest/v2/configuration.html#clustering-flags

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
	fmt.Println("etcd done")

	log.Fatal(<-e.Err())
}
