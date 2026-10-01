/*
Copyright 2021 The Kubernetes Authors.
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
	"strings"
	"testing"
)

// rulesPerIP is the number of iptables rules initIptables adds per local IP.
// Keep in sync with the rule list in initIptables.
const rulesPerIP = 14

// containsIP reports whether any argument of the rule references the given IP.
func ruleReferencesIP(r iptablesRule, ip string) bool {
	for _, arg := range r.args {
		if arg == ip {
			return true
		}
	}
	return false
}

func TestInitIptablesSplitsRulesByFamily(t *testing.T) {
	testCases := []struct {
		name          string
		localIPStr    string
		wantIPv4Rules int
		wantIPv6Rules int
		wantV4Handle  bool
		wantV6Handle  bool
	}{
		{
			name:          "ipv4 single-stack",
			localIPStr:    "169.254.20.10,10.0.0.10",
			wantIPv4Rules: 2 * rulesPerIP,
			wantIPv6Rules: 0,
			wantV4Handle:  true,
			wantV6Handle:  false,
		},
		{
			name:          "ipv6 single-stack",
			localIPStr:    "fd00:1:2:3::5",
			wantIPv4Rules: 0,
			wantIPv6Rules: rulesPerIP,
			wantV4Handle:  false,
			wantV6Handle:  true,
		},
		{
			name:          "dual-stack",
			localIPStr:    "169.254.20.10,fd00:1:2:3::5",
			wantIPv4Rules: rulesPerIP,
			wantIPv6Rules: rulesPerIP,
			wantV4Handle:  true,
			wantV6Handle:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := &CacheApp{params: &ConfigParams{
				LocalIPStr:    tc.localIPStr,
				LocalPort:     "53",
				HealthPort:    "8080",
				SetupIptables: true,
			}}
			c.initIptables()

			if got := len(c.iptablesRules[ipv4]); got != tc.wantIPv4Rules {
				t.Errorf("IPv4 rule count = %d, want %d", got, tc.wantIPv4Rules)
			}
			if got := len(c.iptablesRules[ipv6]); got != tc.wantIPv6Rules {
				t.Errorf("IPv6 rule count = %d, want %d", got, tc.wantIPv6Rules)
			}

			// A handle must be created iff there are rules for that family, so
			// that single-stack nodes never invoke the missing ip(6)tables binary.
			if (c.iptables[ipv4] != nil) != tc.wantV4Handle {
				t.Errorf("IPv4 iptables handle present = %v, want %v", c.iptables[ipv4] != nil, tc.wantV4Handle)
			}
			if (c.iptables[ipv6] != nil) != tc.wantV6Handle {
				t.Errorf("IPv6 iptables handle present = %v, want %v", c.iptables[ipv6] != nil, tc.wantV6Handle)
			}

			// Every rule in a family bucket must reference an IP of that family only.
			for _, r := range c.iptablesRules[ipv4] {
				for _, ip := range strings.Split(tc.localIPStr, ",") {
					if strings.Contains(ip, ":") && ruleReferencesIP(r, ip) {
						t.Errorf("IPv4 bucket contains rule referencing IPv6 address %q: %v", ip, r)
					}
				}
			}
			for _, r := range c.iptablesRules[ipv6] {
				for _, ip := range strings.Split(tc.localIPStr, ",") {
					if !strings.Contains(ip, ":") && ruleReferencesIP(r, ip) {
						t.Errorf("IPv6 bucket contains rule referencing IPv4 address %q: %v", ip, r)
					}
				}
			}
		})
	}
}

func TestIPFamilyOf(t *testing.T) {
	c := &CacheApp{params: &ConfigParams{
		LocalIPStr:    "10.0.0.10",
		LocalPort:     "53",
		HealthPort:    "8080",
		SetupIptables: true,
	}}
	c.initIptables()
	if c.iptables[ipv6] != nil {
		t.Errorf("expected no IPv6 handle for IPv4-only config")
	}
	if len(c.iptablesRules[ipv6]) != 0 {
		t.Errorf("expected no IPv6 rules for IPv4-only config, got %d", len(c.iptablesRules[ipv6]))
	}
}
