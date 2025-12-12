package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SepehrImanian/glbp/internal/adapters/netio"
	"github.com/SepehrImanian/glbp/internal/adapters/repo"
	"github.com/SepehrImanian/glbp/internal/app"
	"github.com/SepehrImanian/glbp/internal/config"
	"github.com/SepehrImanian/glbp/internal/domain"
)

func main() {
	configPath := flag.String("config", "glbp.yaml", "Path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	r := repo.NewMemory()
	hello, err := netio.NewHelloUDP(cfg)
	if err != nil {
		log.Fatalf("hello udp: %v", err)
	}
	arp, err := netio.NewARP(cfg)
	if err != nil {
		log.Fatalf("arp: %v", err)
	}
	info, err := netio.NewLocalInfo(cfg)
	if err != nil {
		log.Fatalf("local info: %v", err)
	}

	d := &app.Daemon{
		GroupID:       uint16(cfg.GroupID),
		VirtualIP:     cfg.VirtualIP,
		LocalPrio:     cfg.Priority,
		LocalWeight:   cfg.Weight,
		Preempt:       cfg.Preempt,
		HelloInterval: time.Second * time.Duration(cfg.HelloTimeSec),
		HoldTime:      time.Second * time.Duration(cfg.HoldTimeSec),
		Hello:         hello,
		ARP:           arp,
		Repo:          r,
		Info:          info,
		Selector:      &domain.RoundRobinSelector{},
		Logger:        log.Default(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		log.Println("shutdown signal received")
		cancel()
	}()

	if err := d.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
