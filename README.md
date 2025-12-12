# glbp

This project is an experimental GLBP-like daemon written in Go for Linux.

It is **not** a Cisco implementation and it is **not wire-compatible with Cisco GLBP**.
The goal is to study and experiment with gateway redundancy, virtual MAC forwarding,
and ARP-based load sharing in a Linux lab.

---

## What it does

- Runs as a user-space daemon on Linux
- Elects one node as an Active Virtual Gateway (AVG) based on priority
- Other nodes act as standby or listen
- Uses multicast hello messages to discover peers
- Replies to ARP requests for a shared virtual IP
- Distributes ARP replies across virtual MAC forwarders (round-robin)
- Uses pcap for ARP receive and transmit

This project is for learning and testing purposes only.

---

## Requirements

- Linux
- Go 1.22 or newer
- Root privileges (pcap and raw packet send)
- libpcap development package

Debian / Ubuntu:

```bash
sudo apt install libpcap-dev
```

---

## Build

From the repository root:

```bash
go mod tidy
go build -o glbpd ./cmd/glbpd
```

---

## Configuration

Example configuration file:

```yaml
interface: eth0
virtual_ip: 10.21.8.10
group_id: 10
priority: 150
weight: 100
preempt: true
hello_time_sec: 3
hold_time_sec: 10
multicast_group: 224.0.0.102
multicast_port: 3222

forwarders:
  - id: 1
    iface: vmac0
```

All routers in the same group must use the same virtual IP, group ID,
and multicast settings.

---

## Network setup example

Create a virtual MAC interface:

```bash
sudo ip link add vmac0 type dummy
sudo ip link set vmac0 address 00:07:b4:00:0a:01
sudo ip link set vmac0 up
sudo ip addr add 10.21.8.10/32 dev vmac0
sudo sysctl -w net.ipv4.ip_forward=1
```

Disable kernel ARP on the LAN interface:

```bash
sudo sysctl -w net.ipv4.conf.eth0.arp_ignore=1
sudo sysctl -w net.ipv4.conf.eth0.arp_announce=2
```

---

## Run

```bash
sudo ./glbpd --config example/glbp-router-1.yaml
```

You should see periodic log output showing the current role,
peer count, and forwarder count.

---

## Limitations

- Not Cisco GLBP compatible
- Simplified hello format
- No authentication
- No production hardening

---

## Legal note

This project is an independent implementation inspired by the behavior of GLBP.
It is not affiliated with or endorsed by Cisco.

Cisco and GLBP are trademarks of Cisco Systems, Inc.