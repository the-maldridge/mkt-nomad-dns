package routeros

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client knows how to authenticate to routeros and send and receive
// queries.
type Client struct {
	username string
	password string
	address  string
	domain   string

	cl *http.Client
}

// DNSRecord matches the minimum information that gets returned from
// the routeros API.
type DNSRecord struct {
	ID      string `json:".id,omitempty"`
	Address string `json:"address,omitempty"`
	Name    string `json:"name,omitempty"`
	Type    string `json:"type,omitempty"`
	Comment string `json:"comment,omitempty"`
}

// New sets up a client pointed at the given address.
func New(address, username, password, domain string) (*Client, error) {
	c := new(Client)
	c.address = address
	c.username = username
	c.password = password
	c.domain = domain

	c.cl = &http.Client{
		Timeout: time.Second * 5,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	return c, nil
}

// ListDNS queries the mikrotik system for DNS and returns a list of
// records that are currently set that have the specified "tag" in the
// comment field.
func (c *Client) ListDNS(tag string) ([]DNSRecord, error) {
	vals := url.Values{}
	vals.Set("comment", tag)
	req := &http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme:   "https",
			Host:     c.address,
			Path:     "/rest/ip/dns/static",
			User:     url.UserPassword(c.username, c.password),
			RawQuery: vals.Encode(),
		},
	}

	resp, err := c.cl.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	rrset := []DNSRecord{}
	if err := json.NewDecoder(resp.Body).Decode(&rrset); err != nil {
		return nil, err
	}

	return rrset, nil
}

// GetRecord retrieves all records for a given name.  Generally this
// list should only be one result, but it could be more.
func (c *Client) GetRecord(key, value string) ([]DNSRecord, error) {
	vals := url.Values{}
	vals.Set(key, value)
	req := &http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme:   "https",
			Host:     c.address,
			Path:     "/rest/ip/dns/static",
			User:     url.UserPassword(c.username, c.password),
			RawQuery: vals.Encode(),
		},
	}

	resp, err := c.cl.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, err
	}
	defer resp.Body.Close()

	rrset := []DNSRecord{}
	if err := json.NewDecoder(resp.Body).Decode(&rrset); err != nil {
		return nil, err
	}

	return rrset, nil
}

// PutRecord creates the specified record.
func (c *Client) PutRecord(rrdata DNSRecord) (DNSRecord, error) {
	req := &http.Request{
		Method: http.MethodPut,
		URL: &url.URL{
			Scheme: "https",
			Host:   c.address,
			Path:   "/rest/ip/dns/static",
			User:   url.UserPassword(c.username, c.password),
		},
	}

	data, err := json.Marshal(rrdata)
	if err != nil {
		return DNSRecord{}, err
	}

	req, err = http.NewRequest(http.MethodPut, req.URL.String(), bytes.NewBuffer(data))
	if err != nil {
		return DNSRecord{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.cl.Do(req)
	if err != nil {
		return DNSRecord{}, fmt.Errorf("Error creating record %s", rrdata.Name)
	}
	if err == nil && resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		fmt.Println(string(b))
		return DNSRecord{}, fmt.Errorf("Error creating record %s", rrdata.Name)
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&rrdata); err != nil {
		return DNSRecord{}, err
	}
	return rrdata, nil
}

// DelRecord removes the specified record.
func (c *Client) DelRecord(rrdata DNSRecord) error {
	req := &http.Request{
		Method: http.MethodDelete,
		URL: &url.URL{
			Scheme: "https",
			Host:   c.address,
			Opaque: "/rest/ip/dns/static/" + rrdata.ID,
			User:   url.UserPassword(c.username, c.password),
		},
	}

	resp, err := c.cl.Do(req)
	defer resp.Body.Close()
	if err != nil || resp.StatusCode != http.StatusOK {
		return err
	}
	return nil
}

// ReconcileDNS takes in a map of strings to addresses for those
// strings.  It will perform the necessary CRUD operations to update
// the DNS as required.
func (c *Client) ReconcileDNS(tag string, records map[string][]string) error {
	seen := make(map[string]struct{})

	for name, addrs := range records {
		rrdatas, _ := c.GetRecord("name", name+"."+c.domain)
		tmp := make(map[string]DNSRecord)
		for _, rec := range rrdatas {
			tmp[rec.Address] = rec
		}
		for _, addr := range addrs {
			if _, set := tmp[addr]; set {
				// This address exists in the
				// records already set, and
				// can be classed as seen.
				seen[tmp[addr].ID] = struct{}{}
				delete(tmp, addr)
				continue
			}

			// Address is not set in any records, add it.
			res, err := c.PutRecord(DNSRecord{
				Name:    name + "." + c.domain,
				Address: addr,
				Comment: tag,
			})
			if err != nil {
				return err
			}
			fmt.Println(res.ID)
			seen[res.ID] = struct{}{}
		}
	}

	remaining, _ := c.GetRecord("comment", tag)

	for _, rec := range remaining {
		if _, keep := seen[rec.ID]; !keep {
			fmt.Println("Delete", rec)
			if err := c.DelRecord(rec); err != nil {
				return err
			}
		}
	}

	return nil
}
