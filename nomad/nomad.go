package nomad

import (
	"github.com/hashicorp/nomad/api"
)

// Client connects to noamd and lists services.
type Client struct {
	*api.Client
}

// New sets up the client
func New() (*Client, error) {
	c := new(Client)

	c.Client, _ = api.NewClient(api.DefaultConfig())

	return c, nil
}

// ListServices retrieves all services across all namespaces currently
// authorized.  This does not support pagination, which is useful to
// implement later, but I don't have enough services to justify this.
func (c *Client) ListServices(tag string) (map[string][]string, error) {
	out := make(map[string][]string)
	res, _, err := c.Namespaces().List(&api.QueryOptions{})
	if err != nil {
		return nil, err
	}
	for _, ns := range res {
		opts := api.QueryOptions{Namespace: ns.Name}
		svcs, _, err := c.Services().List(&opts)
		if err != nil {
			return nil, err
		}

		for _, namespace := range svcs {
			for _, svcStub := range namespace.Services {
				// Automatically disable filtering if the tag isn't set
				filtered := len(tag) != 0
				for _, t := range svcStub.Tags {
					if t == tag {
						filtered = false
					}
				}
				if filtered {
					continue
				}

				res, _, err := c.Services().Get(svcStub.ServiceName, &opts)
				if err != nil {
					return nil, err
				}
				for _, instance := range res {
					out[svcStub.ServiceName] = append(out[svcStub.ServiceName], instance.Address)
				}
			}
		}
	}
	return out, nil
}
