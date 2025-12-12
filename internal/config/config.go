package config

import (
    "fmt"
    "net"
    "os"

    "gopkg.in/yaml.v3"
)

type Config struct {
    Interface      string `yaml:"interface"`
    VirtualIP      string `yaml:"virtual_ip"`
    GroupID        uint8  `yaml:"group_id"`
    Priority       uint8  `yaml:"priority"`
    Weight         uint8  `yaml:"weight"`
    Preempt        bool   `yaml:"preempt"`
    HelloTimeSec   int    `yaml:"hello_time_sec"`
    HoldTimeSec    int    `yaml:"hold_time_sec"`
    MulticastGroup string `yaml:"multicast_group"`
    MulticastPort  int    `yaml:"multicast_port"`
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
    if net.ParseIP(c.VirtualIP) == nil {
        return nil, fmt.Errorf("invalid virtual_ip")
    }
    if c.MulticastGroup == "" {
        c.MulticastGroup = "224.0.0.102"
    }
    if c.MulticastPort == 0 {
        c.MulticastPort = 3222
    }
    if c.HelloTimeSec == 0 {
        c.HelloTimeSec = 3
    }
    if c.HoldTimeSec == 0 {
        c.HoldTimeSec = 10
    }
    if c.Weight == 0 {
        c.Weight = 100
    }

    return &c, nil
}
