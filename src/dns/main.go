// Copyright (c) 2014 The SkyDNS Authors. All rights reserved.
// Use of this source code is governed by The MIT License (MIT) that can be
// found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"

	"github.com/skynetservices/skydns/msg"
	"github.com/skynetservices/skydns/server"
)

var (
	tlskey     = ""
	tlspem     = ""
	cacert     = ""
	config     = &server.Config{ReadTimeout: 0, Domain: "", DnsAddr: "", DNSSEC: ""}
	nameserver = ""
	machine    = ""
	discover   = false
	stub       = false
)

func env(key, def string) string {
	if x := os.Getenv(key); x != "" {
		return x
	}
	return def
}

func init() {
	flag.StringVar(&config.Domain, "domain", env("SKYDNS_DOMAIN", "skydns.local."), "domain to anchor requests to (SKYDNS_DOMAIN)")
	flag.StringVar(&config.DnsAddr, "addr", env("SKYDNS_ADDR", "127.0.0.1:53"), "ip:port to bind to (SKYDNS_ADDR)")
	flag.StringVar(&nameserver, "nameservers", env("SKYDNS_NAMESERVERS", ""), "nameserver address(es) to forward (non-local) queries to e.g. 8.8.8.8:53,8.8.4.4:53")
	flag.BoolVar(&config.NoRec, "no-rec", false, "do not provide a recursive service")
	flag.StringVar(&machine, "machines", env("ETCD_MACHINES", ""), "machine address(es) running etcd")
	flag.StringVar(&config.DNSSEC, "dnssec", "", "basename of DNSSEC key file e.q. Kskydns.local.+005+38250")
	flag.StringVar(&config.Local, "local", "", "optional unique value for this skydns instance")
	flag.StringVar(&tlskey, "tls-key", env("ETCD_TLSKEY", ""), "TLS Private Key path")
	flag.StringVar(&tlspem, "tls-pem", env("ETCD_TLSPEM", ""), "X509 Certificate")
	flag.StringVar(&cacert, "ca-cert", env("ETCD_CACERT", ""), "CA Certificate")
	flag.DurationVar(&config.ReadTimeout, "rtimeout", 2*time.Second, "read timeout")
	flag.BoolVar(&config.RoundRobin, "round-robin", true, "round robin A/AAAA replies")
	flag.BoolVar(&discover, "discover", false, "discover new machines by watching /v2/_etcd/machines")
	flag.BoolVar(&stub, "stubzones", false, "support stub zones")
	flag.BoolVar(&config.Verbose, "verbose", false, "log queries")
	flag.BoolVar(&config.Systemd, "systemd", false, "bind to socket(s) activated by systemd (ignore -addr)")

	// TTl
	// Minttl
	flag.StringVar(&config.Hostmaster, "hostmaster", "hostmaster@skydns.local.", "hostmaster email address to use")
	flag.IntVar(&config.SCache, "scache", server.SCacheCapacity, "capacity of the signature cache")
	flag.IntVar(&config.RCache, "rcache", 0, "capacity of the response cache") // default to 0 for now
	flag.IntVar(&config.RCacheTtl, "rcache-ttl", server.RCacheTtl, "TTL of the response cache")

	flag.StringVar(&msg.PathPrefix, "path-prefix", env("SKYDNS_PATH_PREFIX", "/"), "backend(etcd) path prefix, default: skydns")
}

func main() {
	flag.Parse()

	if nameserver != "" {
		for _, hostPort := range strings.Split(nameserver, ",") {
			if err := validateHostPort(hostPort); err != nil {
				log.Fatalf("skydns: nameserver is invalid: %s", err)
			}
			config.Nameservers = append(config.Nameservers, hostPort)
		}
	}
	if err := validateHostPort(config.DnsAddr); err != nil {
		log.Fatalf("skydns: addr is invalid: %s", err)
	}

	if err := server.SetDefaults(config); err != nil {
		log.Fatalf("skydns defaults could not be set from /etc/resolv.conf: %v", err)
	}

	if config.Local != "" {
		config.Local = dns.Fqdn(config.Local)
	}

	backend := NewBackend()
	s := server.New(backend, config)

	//stats.Collect()  // Graphite
	//server.Metrics() // Prometheus

	if err := s.Run(); err != nil {
		log.Fatalf("skydns: %s", err)
	}
}

func validateHostPort(hostPort string) error {
	host, port, err := net.SplitHostPort(hostPort)
	if err != nil {
		return err
	}
	if ip := net.ParseIP(host); ip == nil {
		return fmt.Errorf("bad IP address: %s", host)
	}

	if p, _ := strconv.Atoi(port); p < 1 || p > 65535 {
		return fmt.Errorf("bad port number %s", port)
	}
	return nil
}
