package main

import (
    "context"
    "flag"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "example.com/glbpd/internal/config"
    "example.com/glbpd/internal/core"
    glbpio "example.com/glbpd/internal/io"
)

func main() {
    cfgPath := flag.String("config", "/etc/glbpd.yaml", "Path to config file")
    flag.Parse()

    cfg, err := config.Load(*cfgPath)
    if err != nil {
        log.Fatalf("load config: %v", err)
    }

    group, err := core.NewGroup(cfg)
    if err != nil {
        log.Fatalf("new group: %v", err)
    }

    helloIO, err := glbpio.NewHelloIO(cfg.Interface, cfg.MulticastGroup, cfg.MulticastPort, group)
    if err != nil {
        log.Fatalf("hello IO: %v", err)
    }
    helloIO.SetFromConfig(cfg)

    arpIO, err := glbpio.NewARPResponder(cfg.Interface, cfg.VirtualIP, group)
    if err != nil {
        log.Fatalf("ARP responder: %v", err)
    }

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Signal handling
    go func() {
        sigCh := make(chan os.Signal, 1)
        signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
        <-sigCh
        log.Printf("signal received, shutting down")
        cancel()
    }()

    // Start hello RX/TX
    go func() {
        if err := helloIO.Run(ctx); err != nil {
            log.Printf("hello IO stopped: %v", err)
            cancel()
        }
    }()

    // Start ARP responder
    go func() {
        if err := arpIO.Run(ctx); err != nil {
            log.Printf("ARP responder stopped: %v", err)
            cancel()
        }
    }()

    // Periodic timers to drive group logic
    ticker := time.NewTicker(500 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case now := <-ticker.C:
            group.OnTimerTick(now)
        }
    }
}
