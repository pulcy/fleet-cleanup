// Copyright (c) 2016 Pulcy.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"path"

	"github.com/coreos/etcd/client"
	"github.com/op/go-logging"
	"golang.org/x/net/context"
)

type ServiceConfig struct {
	EtcdURL url.URL
	DryRun  bool
}

type ServiceDependencies struct {
	Logger *logging.Logger
}

type Service struct {
	ServiceConfig
	ServiceDependencies

	client client.Client
}

type jobObject struct {
	Name     string `json:"Name"`
	UnitHash []byte `json:"UnitHash"`
}

func (j jobObject) Hash() string {
	return hex.EncodeToString(j.UnitHash)
}

// NewService creates a new service instance.
func NewService(config ServiceConfig, deps ServiceDependencies) (*Service, error) {
	cfg := client.Config{
		Transport: client.DefaultTransport,
	}
	if config.EtcdURL.Host != "" {
		cfg.Endpoints = append(cfg.Endpoints, "http://"+config.EtcdURL.Host)
	}
	c, err := client.New(cfg)
	if err != nil {
		return nil, maskAny(err)
	}
	s := &Service{
		ServiceConfig:       config,
		ServiceDependencies: deps,
		client:              c,
	}
	return s, nil
}

// Run performs a single cleanup
func (s *Service) Run() error {
	// Load unit names (hex)
	unitHashes, err := s.loadUnitNames()
	if err != nil {
		return maskAny(err)
	}

	// Load job objects
	objects, err := s.loadObjects()
	if err != nil {
		return maskAny(err)
	}

	// Derive valid hashes
	validHashes := make(map[string]jobObject)
	for _, j := range objects {
		validHashes[j.Hash()] = j
	}

	// Remove obsolete units
	keysAPI := client.NewKeysAPI(s.client)
	removed := 0
	for _, unit := range unitHashes {
		if _, ok := validHashes[unit]; ok {
			continue
		}
		// Found obsolete unit
		key := fmt.Sprintf("/_coreos.com/fleet/unit/%s", unit)
		if s.DryRun {
			s.Logger.Infof("Obsolete unit at %s", key)
		} else {
			s.Logger.Infof("Removing obsolete unit at %s", key)
			if _, err := keysAPI.Delete(context.Background(), key, &client.DeleteOptions{}); err != nil {
				s.Logger.Errorf("Failed to remove obsolete unit at %s: %#v", key, err)
				return maskAny(err)
			}
			removed++
		}
	}

	if s.DryRun {
		s.Logger.Infof("Found %d jobs, %d obsolete units can be removed", len(objects), removed)
	} else {
		s.Logger.Infof("Found %d jobs, removed %d obsolete units", len(objects), removed)
	}
	return nil
}

// Load all unit names stored by fleet
func (s *Service) loadUnitNames() ([]string, error) {
	keysAPI := client.NewKeysAPI(s.client)

	// Load unit names (hex)
	resp, err := keysAPI.Get(context.Background(), "/_coreos.com/fleet/unit", &client.GetOptions{})
	if err != nil {
		return nil, maskAny(err)
	}

	result := []string{}
	if resp.Node != nil {
		for _, n := range resp.Node.Nodes {
			name := path.Base(n.Key)
			result = append(result, name)
		}
	}
	return result, nil
}

// Load all job objects stored by fleet
func (s *Service) loadObjects() ([]jobObject, error) {
	keysAPI := client.NewKeysAPI(s.client)

	// Load unit names (hex)
	resp, err := keysAPI.Get(context.Background(), "/_coreos.com/fleet/job", &client.GetOptions{Recursive: true})
	if err != nil {
		return nil, maskAny(err)
	}

	result := []jobObject{}
	if resp.Node != nil {
		// For over jobs
		for _, n := range resp.Node.Nodes {
			// Find object
			for _, c := range n.Nodes {
				name := path.Base(c.Key)
				if name != "object" {
					continue
				}
				// found object, parse it
				raw := c.Value
				var data jobObject
				if err := json.Unmarshal([]byte(raw), &data); err != nil {
					s.Logger.Errorf("Failed to parse '%s': %#v", raw, err)
					return nil, maskAny(err)
				}
				result = append(result, data)
			}
		}
	}
	return result, nil
}
