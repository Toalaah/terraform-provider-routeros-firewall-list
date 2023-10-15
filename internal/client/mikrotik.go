/*
 * terraform-provider-routeros-firewall-list
 * Copyright (C) 2023  Samuel Kunst
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package client

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type Client struct {
	hostURL  string
	username string
	password string
	client   *http.Client
}

type FirewallRule struct {
	ID    string `json:".id"`
	Chain string `json:"chain"`
	Next  *FirewallRule
}

type ClientOpts struct {
	HostURL  string
	Username string
	Password string
	CA       string
	Insecure bool
}

func New(opts ClientOpts) (*Client, error) {
	if opts.CA == "" {
		return nil, errors.New("No CA cert provided")
	}

	if _, err := os.Stat(opts.CA); err != nil {
		return nil, fmt.Errorf("Could not open file at provided path %s\n", opts.CA)
	}

	certPool := x509.NewCertPool()
	file, err := os.ReadFile(opts.CA)
	if err != nil {
		return nil, fmt.Errorf("Could not read file at provided path %s\n", opts.CA)
	}

	certPool.AppendCertsFromPEM(file)

	tls := &tls.Config{
		InsecureSkipVerify: opts.Insecure,
		RootCAs:            certPool,
	}

	return &Client{
		hostURL:  opts.HostURL,
		username: opts.Username,
		password: opts.Password,
		client: &http.Client{
			Transport: &http.Transport{TLSClientConfig: tls},
		},
	}, nil
}

func basicAuth(username, password string) string {
	auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	return fmt.Sprintf("Basic %s", auth)
}

func (c *Client) MakeRequest(method, cmd string, body []byte) (*http.Response, error) {
	var (
		req *http.Request
		err error
		url string = fmt.Sprintf("%s/rest/%s", c.hostURL, strings.TrimPrefix(cmd, "/"))
	)

	if body == nil {
		req, err = http.NewRequest(method, url, nil)
	} else {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(body))
	}
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", basicAuth(c.username, c.password))
	req.Header.Add("Content-Type", "application/json")

	return c.client.Do(req)
}

func (c *Client) GetOrderingFrom(ruleType string, start FirewallRule, length int) ([]FirewallRule, error) {
	var ordering []FirewallRule

	ordering, err := c.GetRulesOfType(ruleType)
	if err != nil {
		return ordering, err
	}

	return ordering, nil
}

// TODO: allow for gaps in subsequence. So if real_state=[1,2,X,3,4] and
// desired_state=[1,2,3,4], this should still return true. Or make a resource
// option to allow for toggling between these two behaviors?
func (c *Client) RuleOrderExists(ruleType string, seq []FirewallRule) (bool, error) {
	var subSeq string
	var ruleSequenceStr string

	for _, rule := range seq {
		subSeq += rule.ID
	}

	rules, err := c.GetRulesOfType(ruleType)
	if err != nil {
		return false, err
	}

	for _, rule := range rules {
		ruleSequenceStr += rule.ID
	}

	return strings.Contains(ruleSequenceStr, subSeq), nil
}

func (c *Client) GetRulesOfType(ruleType string) ([]FirewallRule, error) {
	rules := []FirewallRule{}

	r, err := c.MakeRequest(http.MethodGet, fmt.Sprintf("/ip/firewall/%s", ruleType), nil)
	if err != nil {
		return rules, err
	}

	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return rules, err
	}

	err = json.Unmarshal(body, &rules)
	if err != nil {
		return rules, err
	}

	for i := range rules {
		if i+1 < len(rules) {
			rules[i].Next = &rules[i+1]
		}
	}

	return rules, nil
}

func (c *Client) GetRule(ruleType, id string) (FirewallRule, error) {
	// Yes, we can also just call the GET endpoint for a single rule, but since
	// we want to augment the return value with the `Next` firewall rule, we need
	// to be able to easily lookup the rule's follower. The console *does* expose
	// the `.nextid` field, however this is not available from in the REST API.
	// And although you can get the output of arbitrary console, parsing it back
	// into a usable struct is a pain.
	rules, err := c.GetRulesOfType(ruleType)
	if err != nil {
		return FirewallRule{}, fmt.Errorf("unable to find rule of type '%s' with id: '%s'", ruleType, id)
	}
	for _, rule := range rules {
		if rule.ID == id {
			return rule, nil
		}
	}
	return FirewallRule{}, fmt.Errorf("unable to find rule of type '%s' with id: '%s'", ruleType, id)
}

func (c *Client) OrderRules(ruleType string, rs ...FirewallRule) error {
	ids := []string{}
	for _, v := range rs {
		ids = append(ids, v.ID)
	}
	payload := strings.Join(ids, ",")

	b := []byte(fmt.Sprintf(`{"numbers":"%s","destination":"%s"}`, payload, "*ffffff"))
	_, err := c.MakeRequest(http.MethodPost, fmt.Sprintf("/ip/firewall/%s/move", ruleType), b)
	return err
}
