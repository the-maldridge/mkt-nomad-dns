package main

import (
	"fmt"
	"os"

	"github.com/the-maldridge/mkt-nomad-dns/nomad"
	"github.com/the-maldridge/mkt-nomad-dns/routeros"
)

func main() {
	n, err := nomad.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing nomad client: %s\n", err)
		return
	}

	svcs, err := n.ListServices(os.Getenv("NOMAD_TAG"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing services: %s\n", err)
		return
	}

	r, err := routeros.New(
		os.Getenv("ROS_ADDRESS"),
		os.Getenv("ROS_USERNAME"),
		os.Getenv("ROS_PASSWORD"),
		os.Getenv("DNS_DOMAIN"),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing routeros client: %s\n", err)
		return
	}

	if err := r.ReconcileDNS(os.Getenv("NOMAD_TAG"), svcs); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating DNS: %s\n", err)
		return
	}
}
