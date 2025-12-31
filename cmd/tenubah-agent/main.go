package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/kardianos/service"

	"github.com/TenubahDEV/tenubah-agent/internal/agent"
	"github.com/TenubahDEV/tenubah-agent/internal/config"
	"github.com/TenubahDEV/tenubah-agent/internal/version"
)

type program struct {
	agent *agent.Agent
	quit  chan struct{}
}

func (p *program) Start(s service.Service) error {
	p.quit = make(chan struct{})

	go p.metricsLoop()
	go p.updateLoop()

	return nil
}

func (p *program) run() {
	updateTicker := time.NewTicker(24 * time.Hour)
	defer updateTicker.Stop()

	for {
		select {
		case <-p.quit:
			return

		case <-updateTicker.C:
			p.agent.CheckUpdate()

		default:
			if err := p.agent.RunOnce(); err != nil {
				log.Println("push error:", err)
			}
			time.Sleep(time.Duration(p.agent.Interval()) * time.Second)
		}
	}
}

func (p *program) Stop(s service.Service) error {
	close(p.quit)
	return nil
}

func (p *program) metricsLoop() {
	for {
		select {
		case <-p.quit:
			return
		default:
			_ = p.agent.RunOnce()
			time.Sleep(time.Duration(p.agent.Interval()) * time.Second)
		}
	}
}

func (p *program) updateLoop() {
	if !p.agent.AutoUpdateEnabled() {
		return
	}

	ticker := time.NewTicker(
		time.Duration(p.agent.UpdateInterval()) * time.Hour,
	)
	defer ticker.Stop()

	for {
		select {
		case <-p.quit:
			return
		case <-ticker.C:
			p.agent.CheckUpdate()
		}
	}
}

func defaultConfigPath() string {
	if v := os.Getenv("TENUBAH_CONFIG"); v != "" {
		return v
	}
	return filepath.Join(".", "config.yaml")
}

func main() {

	cfgPath := flag.String("config", defaultConfigPath(), "config file")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Tenubah Agent starting, version=%s", version.Version)
	ag := agent.New(cfg)

	svcConfig := &service.Config{
		Name:        "tenubah-agent",
		DisplayName: "Tenubah NOC Agent",
		Description: "Tenubah monitoring agent pushing metrics to PushGateway",

		Arguments: []string{
			"-config",
			*cfgPath,
		},
	}
	showVersion := flag.Bool("version", false, "show version")
	flag.Parse()

	if *showVersion {
		log.Println("Tenubah Agent version:", version.Version)
		return
	}
	disableUpdate := flag.Bool("disable-auto-update", false, "disable auto update")
	flag.Parse()

	if *disableUpdate {
		cfg.AutoUpdate.Enabled = false
	}

	prg := &program{agent: ag}
	svc, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	if len(flag.Args()) > 0 {
		if err := service.Control(svc, flag.Args()[0]); err != nil {
			log.Fatal(err)
		}
		return
	}

	err = svc.Run()
	if err != nil {
		log.Fatal(err)
	}
}
