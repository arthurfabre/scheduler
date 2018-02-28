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
func URL(ip string, port uint16) []url.URL {
	// TODO - We probably shouldn't ignore the error?
	parsed, _ := url.Parse(fmt.Sprintf("http://%s:%d", ip, port))
	return []url.URL{*parsed}
}

func startEtcd(ip string, clientPort uint16, peerPort uint16) {
	cfg := embed.NewConfig()

	// TODO - Is this sane?
	cfg.Dir = "default.etcd"

	// Other etcd servers
	cfg.LPUrls = URL(ip, peerPort)
	// Client
	cfg.LCUrls = URL(ip, clientPort)

	// We only bing the public IP, advertise that
	cfg.APUrls = cfg.LPUrls
	cfg.ACUrls = cfg.LCUrls

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
