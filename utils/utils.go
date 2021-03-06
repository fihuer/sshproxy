// Copyright 2015-2017 CEA/DAM/DIF
//  Contributor: Arnaud Guignard <arnaud.guignard@cea.fr>
//
// This software is governed by the CeCILL-B license under French law and
// abiding by the rules of distribution of free software.  You can  use,
// modify and/ or redistribute the software under the terms of the CeCILL-B
// license as circulated by CEA, CNRS and INRIA at the following URL
// "http://www.cecill.info".

package utils

import (
	"crypto/sha1"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/cea-hpc/sshproxy/group.go"
)

const DefaultSshPort = "22"

// CalcSessionId returns a unique 10 hexadecimal characters string from
// a user name, time, ip address and port.
func CalcSessionId(user string, t time.Time, hostport string) string {
	sum := sha1.Sum([]byte(fmt.Sprintf("%s@%s@%d", user, hostport, t.UnixNano())))
	return fmt.Sprintf("%X", sum[:5])
}

// SplitHostPort splits a network address of the form "host:port" or
// "host[:port]" into host and port. If the port is not specified the default
// ssh port ("22") is returned.
func SplitHostPort(hostport string) (string, string, error) {
	host, port, err := net.SplitHostPort(hostport)
	if err != nil {
		if err.(*net.AddrError).Err == "missing port in address" {
			return hostport, DefaultSshPort, nil
		} else {
			return hostport, DefaultSshPort, err
		}
	}
	return host, port, nil
}

// GetGroups returns a map of group memberships for the current user.
//
// It can be used to quickly check if a user is in a specified group.
func GetGroups() (map[string]bool, error) {
	gids, err := os.Getgroups()
	if err != nil {
		return nil, err
	}

	groups := make(map[string]bool)
	for _, gid := range gids {
		g, err := group.LookupId(gid)
		if err != nil {
			return nil, err
		}

		groups[g.Name] = true
	}

	return groups, nil
}

// CheckRoutes checks and replaces all hosts defined in a map of routes with
// their "host:port" value (in case the host is defined without a port).
func CheckRoutes(routes map[string][]string) error {
	for source, destinations := range routes {
		host, port, err := SplitHostPort(source)
		if err != nil {
			return fmt.Errorf("invalid source address '%s': %s", source, err)
		}

		full_source := net.JoinHostPort(host, port)
		delete(routes, source)

		for i, dst := range destinations {
			host, port, err := SplitHostPort(dst)
			if err != nil {
				return fmt.Errorf("invalid destination '%s' for source address '%s': %s", dst, source, err)
			}
			destinations[i] = net.JoinHostPort(host, port)
		}
		routes[full_source] = destinations
	}
	return nil
}
