sudo ip link add vmac0 type dummy
sudo ip link set vmac0 address 00:07:b4:00:0a:02
sudo ip link set vmac0 up
sudo ip addr add 10.21.8.10/32 dev vmac0
sudo sysctl -w net.ipv4.ip_forward=1
sudo sysctl -w net.ipv4.conf.eth0.arp_ignore=1
sudo sysctl -w net.ipv4.conf.eth0.arp_announce=2