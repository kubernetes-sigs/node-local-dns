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

package util

import (
	"fmt"
	"net"
	"strconv"
)

// ValidateNameserverIpAndPort splits and validates ip and port for nameserver.
// If there is no port in the given address, a default 53 port will be returned.
func ValidateNameserverIpAndPort(nameServer string) (string, string, error) {
	if ip := net.ParseIP(nameServer); ip != nil {
		return ip.String(), "53", nil
	}

	host, port, err := net.SplitHostPort(nameServer)
	if err != nil {
		return "", "", err
	}
	if ip := net.ParseIP(host); ip == nil {
		return "", "", fmt.Errorf("bad IP address: %q", host)
	}
	if p, err := strconv.Atoi(port); err != nil || p < 1 || p > 65535 {
		return "", "", fmt.Errorf("bad port number: %q", port)
	}
	return host, port, nil
}
