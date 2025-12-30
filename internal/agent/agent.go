package agent

import (
	"log"
	"os"

	"github.com/TenubahDEV/tenubah-agent/internal/collectors"
	"github.com/TenubahDEV/tenubah-agent/internal/config"
	"github.com/TenubahDEV/tenubah-agent/internal/pusher"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/TenubahDEV/tenubah-agent/internal/updater"
	"github.com/TenubahDEV/tenubah-agent/internal/version"
)

type Agent struct {
	cfg *config.Config
}

func New(cfg *config.Config) *Agent {
	return &Agent{cfg: cfg}
}

func (a *Agent) instance() string {
	if a.cfg.InstanceName != "" {
		return a.cfg.InstanceName
	}
	h, _ := os.Hostname()
	return h
}

func (a *Agent) RunOnce() error {
	ctx, cancel := collectors.Ctx()
	defer cancel()

	m, err := collectors.Collect(ctx)
	if err != nil {
		return err
	}

	// 🔒 Registry dedicado (CLAVE)
	reg := prometheus.NewRegistry()

	cpu := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tenubah_cpu_usage_percent",
		Help: "CPU usage percentage",
	})
	mem := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tenubah_memory_used_mb",
		Help: "Used memory in MB",
	})
	disk := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tenubah_disk_usage_percent",
		Help: "Disk usage percent",
	})
	uptime := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tenubah_uptime_seconds",
		Help: "System uptime in seconds",
	})
	load1 := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tenubah_load1",
		Help: "Load average 1m",
	})

	// Registrar SIEMPRE
	reg.MustRegister(cpu, mem, disk, uptime, load1)

	// Set values
	cpu.Set(m.CPU)
	mem.Set(m.MemUsed)
	disk.Set(m.DiskUsedPercent)
	uptime.Set(float64(m.Uptime))
	load1.Set(m.Load1)

	pc := pusher.Client{
		URL:   a.cfg.PushgatewayURL,
		Token: a.cfg.Token,
	}

	// 🔥 Push usando Gatherer
	return pc.PushGatherer(
		a.cfg.JobName,
		a.instance(),
		a.cfg.Labels,
		reg,
	)

}

func (a *Agent) Interval() int {
	return a.cfg.IntervalSeconds
}

func (a *Agent) CheckUpdate() {
	if !a.cfg.AutoUpdate.Enabled {
		return
	}

	newVer, url, sigURL, err := updater.CheckLatest(version.Version)
	if err != nil || newVer == "" {
		return
	}

	log.Println("Auto-update: upgrading to", newVer)

	if err := updater.Apply(url, sigURL); err == nil {
		os.Exit(0) // service manager lo reinicia
	}
}

func (a *Agent) AutoUpdateEnabled() bool {
	return a.cfg.AutoUpdate.Enabled
}

func (a *Agent) UpdateInterval() int {
	return a.cfg.AutoUpdate.CheckIntervalHours
}
