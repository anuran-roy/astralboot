package main

import (
	"bytes"
	"flag"
	"fmt"
	"net"
	"text/template"

	"github.com/BurntSushi/toml"
)

type operatingSystem struct {
	Name        string
	Description string
	templates   *template.Template
}

type Config struct {
	Ref       string `toml:"ref"`
	Interf    string `toml:"interface"`
	BaseIP    net.IP
	Gateway   net.IP
	DNSServer net.IP
	DBname    string
	// not exported generated config parts
	fs     ROfs
	OSList map[string]*operatingSystem
}

func GetConfig(path string) (c *Config) {
	if _, err := toml.DecodeFile(path, &c); err != nil {
		logger.Critical("Config file does not exists,create config")
		return
	}
	// bind the cache (not exported)
	// Add items from system not in config file
	if c.Interf == "" {
		c.Interf = "eth0"
	}
	interf, err := net.InterfaceByName(c.Interf)
	if err != nil {
		logger.Critical("Interface error ", err)
	}
	addressList, _ := interf.Addrs()
	serverAddress, _, _ := net.ParseCIDR(addressList[0].String())
	logger.Critical("Server Address  : %s", serverAddress)
	c.BaseIP = serverAddress
	if c.Gateway == nil {
		c.Gateway = serverAddress
	}
	if c.DNSServer == nil {
		c.DNSServer = serverAddress
	}
	// database file name
	if c.DBname == "" {
		c.DBname = "./leases.db"
	}
	//TODO select file system from flag or config

	fileFlag := flag.Bool("l", false, "Use local file sytem")
	var filesystem ROfs
	flag.Parse()
	if *fileFlag {
		filesystem = &Diskfs{"./data"}
	} else {
		filesystem = &IPfsfs{c.Ref}
	}
	c.fs = filesystem

	// distributions
	c.OSList = c.OSListGet()

	return
}

func (c *Config) PrintConfig() {
	buf := new(bytes.Buffer)
	err := toml.NewEncoder(buf).Encode(c)
	fmt.Println(buf.String(), err)
}
