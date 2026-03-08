/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package app

import (
	"time"

	"github.com/spf13/pflag"
)

type NodeCacheConfig struct {
	ConfigDir    string
	ConfigPeriod time.Duration
}

func NewNodeCacheConfig() *NodeCacheConfig {
	return &NodeCacheConfig{
		ConfigPeriod: 10 * time.Second,
		ConfigDir:    "",
	}
}

func (s *NodeCacheConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.ConfigDir, "config-dir", s.ConfigDir,
		"directory to read config values from. Cannot be "+
			"used in conjunction with federations or config-map flag.")
	fs.DurationVar(&s.ConfigPeriod, "config-period", s.ConfigPeriod,
		"period at which to check for updates in config-dir.")
}
