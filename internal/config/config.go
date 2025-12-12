package config

import (
	"fmt"
	"net"
	"os"

	"gopkg.in/yaml.v3"
)

type Forwarder struct {
	ID     uint8  `yaml:"id"`
	Iface  string `yaml:"iface"`
	Weight uint8  `yaml:"weight"`
}

type Config struct {
	Interface      string `yaml:"interface"`
	VirtualIP      net.IP `yaml:"virtual_ip"`
	GroupID        uint8  `yaml:"group_id"`
	Priority       uint8  `yaml:"priority"`
	Weight         uint8  `yaml:"weight"`
	Preempt        bool   `yaml:"preempt"`
	HelloTimeSec   int    `yaml:"hello_time_sec"`
	HoldTimeSec    int    `yaml:"hold_time_sec"`
	MulticastGroup string `yaml:"multicast_group"`
	MulticastPort  int    `yaml:"multicast_port"`
	Forwarders     []Forwarder `yaml:"forwarders"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}

	if c.Interface == "" {
		return nil, fmt.Errorf("interface is required")
	}
	if ip4 := c.VirtualIP.To4(); ip4 == nil {
		return nil, fmt.Errorf("virtual_ip must be IPv4")
	}
	c.VirtualIP = c.VirtualIP.To4()

	if c.MulticastGroup == "" {
		c.MulticastGroup = "224.0.0.102"
	}
	if c.MulticastPort == 0 {
		c.MulticastPort = 3222
	}
	if c.HelloTimeSec <= 0 {
		c.HelloTimeSec = 3
	}
	if c.HoldTimeSec <= 0 {
		c.HoldTimeSec = 10
	}
	if c.Weight == 0 {
		c.Weight = 100
	}
	for i := range c.Forwarders {
		if c.Forwarders[i].ID == 0 {
			return nil, fmt.Errorf("forwarders[%d].id must be set", i)
		}
		if c.Forwarders[i].Iface == "" {
			return nil, fmt.Errorf("forwarders[%d].iface must be set", i)
		}
		if c.Forwarders[i].Weight == 0 {
			c.Forwarders[i].Weight = 100
		}
	}

	return &c, nil
}
