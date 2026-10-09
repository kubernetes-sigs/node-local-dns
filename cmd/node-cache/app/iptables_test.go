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
	"net"
	"strings"
	"testing"

	utiliptables "k8s.io/kubernetes/pkg/util/iptables"
	utilnet "k8s.io/utils/net"
)

// rulesPerIP is the number of iptables rules initIptables adds per local IP.
// Keep in sync with the rule list in initIptables.
const rulesPerIP = 12

// containsIP reports whether any argument of the rule references the given IP.
func ruleReferencesIP(r iptablesRule, ip string) bool {
	for _, arg := range r.args {
		if arg == ip {
			return true
		}
	}
	return false
}

// parseTestIPs parses a comma-separated list of IPs for use in test setup.
func parseTestIPs(t *testing.T, localIPStr string) []net.IP {
	t.Helper()
	var ips []net.IP
	for _, s := range strings.Split(localIPStr, ",") {
		ip := net.ParseIP(s)
		if ip == nil {
			t.Fatalf("invalid test IP %q", s)
		}
		ips = append(ips, ip)
	}
	return ips
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
				LocalIPs:      parseTestIPs(t, tc.localIPStr),
				LocalPort:     "53",
				HealthPort:    "8080",
				SetupIptables: true,
			}}
			c.initIptables()

			if got := len(c.iptablesRules[utilnet.IPv4]); got != tc.wantIPv4Rules {
				t.Errorf("IPv4 rule count = %d, want %d", got, tc.wantIPv4Rules)
			}
			if got := len(c.iptablesRules[utilnet.IPv6]); got != tc.wantIPv6Rules {
				t.Errorf("IPv6 rule count = %d, want %d", got, tc.wantIPv6Rules)
			}

			// A handle must be created iff there are rules for that family, so
			// that single-stack nodes never invoke the missing ip(6)tables binary.
			if (c.iptables[utilnet.IPv4] != nil) != tc.wantV4Handle {
				t.Errorf("IPv4 iptables handle present = %v, want %v", c.iptables[utilnet.IPv4] != nil, tc.wantV4Handle)
			}
			if (c.iptables[utilnet.IPv6] != nil) != tc.wantV6Handle {
				t.Errorf("IPv6 iptables handle present = %v, want %v", c.iptables[utilnet.IPv6] != nil, tc.wantV6Handle)
			}

			// Every rule in a family bucket must reference an IP of that family only.
			for _, r := range c.iptablesRules[utilnet.IPv4] {
				for _, ip := range strings.Split(tc.localIPStr, ",") {
					if strings.Contains(ip, ":") && ruleReferencesIP(r, ip) {
						t.Errorf("IPv4 bucket contains rule referencing IPv6 address %q: %v", ip, r)
					}
				}
			}
			for _, r := range c.iptablesRules[utilnet.IPv6] {
				for _, ip := range strings.Split(tc.localIPStr, ",") {
					if !strings.Contains(ip, ":") && ruleReferencesIP(r, ip) {
						t.Errorf("IPv6 bucket contains rule referencing IPv4 address %q: %v", ip, r)
					}
				}
			}
		})
	}
}

// fakeIPTables records the rules passed to EnsureRule and DeleteRule so that
// setupNetworking and TeardownNetworking can be exercised without touching the
// host's iptables.
type fakeIPTables struct {
	utiliptables.Interface
	ensured []iptablesRule
	deleted []iptablesRule
}

func (f *fakeIPTables) EnsureRule(_ utiliptables.RulePosition, table utiliptables.Table, chain utiliptables.Chain, args ...string) (bool, error) {
	f.ensured = append(f.ensured, iptablesRule{table: table, chain: chain, args: args})
	return false, nil
}

func (f *fakeIPTables) DeleteRule(table utiliptables.Table, chain utiliptables.Chain, args ...string) error {
	f.deleted = append(f.deleted, iptablesRule{table: table, chain: chain, args: args})
	return nil
}

func TestSetupAndTeardownNetworkingDualStack(t *testing.T) {
	localIPStr := "169.254.20.10,fd00:1:2:3::5"
	c := &CacheApp{params: &ConfigParams{
		LocalIPStr:    localIPStr,
		LocalIPs:      parseTestIPs(t, localIPStr),
		LocalPort:     "53",
		HealthPort:    "8080",
		SetupIptables: true,
	}}
	c.initIptables()

	v4Fake := &fakeIPTables{}
	v6Fake := &fakeIPTables{}
	c.iptables[utilnet.IPv4] = v4Fake
	c.iptables[utilnet.IPv6] = v6Fake

	c.setupNetworking()
	if len(v4Fake.ensured) != rulesPerIP {
		t.Errorf("IPv4 EnsureRule calls = %d, want %d", len(v4Fake.ensured), rulesPerIP)
	}
	if len(v6Fake.ensured) != rulesPerIP {
		t.Errorf("IPv6 EnsureRule calls = %d, want %d", len(v6Fake.ensured), rulesPerIP)
	}

	if err := c.TeardownNetworking(); err != nil {
		t.Errorf("TeardownNetworking() unexpected error: %v", err)
	}

	if len(v4Fake.deleted) != 2*rulesPerIP {
		t.Errorf("IPv4 DeleteRule calls = %d, want %d", len(v4Fake.deleted), 2*rulesPerIP)
	}
	if len(v6Fake.deleted) != 2*rulesPerIP {
		t.Errorf("IPv6 DeleteRule calls = %d, want %d", len(v6Fake.deleted), 2*rulesPerIP)
	}
}

func TestParseLocalIPs(t *testing.T) {
	testCases := []struct {
		name         string
		localIPStr   string
		wantErr      bool
		wantFamilies []utilnet.IPFamily
	}{
		{
			name:         "ipv4 single-stack",
			localIPStr:   "169.254.20.10,10.0.0.10",
			wantFamilies: []utilnet.IPFamily{utilnet.IPv4, utilnet.IPv4},
		},
		{
			name:         "ipv6 single-stack",
			localIPStr:   "fd00:1:2:3::5",
			wantFamilies: []utilnet.IPFamily{utilnet.IPv6},
		},
		{
			name:         "dual-stack",
			localIPStr:   "169.254.20.10,fd00:1:2:3::5",
			wantFamilies: []utilnet.IPFamily{utilnet.IPv4, utilnet.IPv6},
		},
		{
			name:       "invalid ip rejected",
			localIPStr: "169.254.20.10,invalid",
			wantErr:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ips, err := ParseLocalIPs(tc.localIPStr)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseLocalIPs(%q) = nil error, want error", tc.localIPStr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseLocalIPs(%q) unexpected error: %v", tc.localIPStr, err)
			}
			if len(ips) != len(tc.wantFamilies) {
				t.Fatalf("ParseLocalIPs(%q) returned %d IPs, want %d", tc.localIPStr, len(ips), len(tc.wantFamilies))
			}
			for i, ip := range ips {
				if got := utilnet.IPFamilyOf(ip); got != tc.wantFamilies[i] {
					t.Errorf("IP %d (%s) family = %v, want %v", i, ip, got, tc.wantFamilies[i])
				}
			}
		})
	}
}
