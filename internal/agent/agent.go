package agent

import (
	"log"
	"os"

	"github.com/TenubahDEV/mexpress_agent/internal/collectors"
	"github.com/TenubahDEV/mexpress_agent/internal/config"
	"github.com/TenubahDEV/mexpress_agent/internal/pusher"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/TenubahDEV/mexpress_agent/internal/updater"
	"github.com/TenubahDEV/mexpress_agent/internal/version"
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
	memPercent := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tenubah_memory_usage_percent",
		Help: "Memory usage percentage",
	})
	swapTotal := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tenubah_swap_total_mb",
		Help: "Swap total in MB",
	})
	swapUsed := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tenubah_swap_used_mb",
		Help: "Swap used in MB",
	})
	diskTotal := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tenubah_disk_total_gb",
		Help: "Disk total in GB",
	})
	diskUsed := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tenubah_disk_used_gb",
		Help: "Disk used in GB",
	})
	diskPercent := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tenubah_disk_usage_percent",
		Help: "Disk usage percent",
	})
	netRx := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tenubah_net_rx_bytes",
		Help: "Network RX bytes",
	})
	netTx := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tenubah_net_tx_bytes",
		Help: "Network TX bytes",
	})
	uptime := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tenubah_uptime_seconds",
		Help: "System uptime in seconds",
	})
	load1 := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tenubah_load1",
		Help: "Load average 1m",
	})
	users := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tenubah_logged_in_users",
		Help: "Number of logged in users",
	})
	procsTotal := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tenubah_processes_total",
		Help: "Total number of processes",
	})
	procsRunning := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tenubah_processes_running",
		Help: "Number of running processes",
	})
	heartbeat := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tenubah_agent_heartbeat",
		Help: "Agent heartbeat (always 1)",
	})

	// Registrar SIEMPRE
	reg.MustRegister(
		cpu,
		mem, memPercent,
		swapTotal, swapUsed,
		diskTotal, diskUsed, diskPercent,
		netRx, netTx,
		uptime, load1,
		users, procsTotal, procsRunning,
		heartbeat,
	)

	// Set values
	cpu.Set(m.CPU)
	mem.Set(m.MemUsed)
	memPercent.Set(m.MemUsedPercent)
	swapTotal.Set(m.SwapTotal)
	swapUsed.Set(m.SwapUsed)
	diskTotal.Set(m.DiskTotal)
	diskUsed.Set(m.DiskUsed)
	diskPercent.Set(m.DiskUsedPercent)
	netRx.Set(float64(m.RxBytes))
	netTx.Set(float64(m.TxBytes))
	uptime.Set(float64(m.Uptime))
	load1.Set(m.Load1)
	users.Set(float64(m.LoggedInUsers))
	procsTotal.Set(float64(m.TotalProcesses))
	procsRunning.Set(float64(m.RunningProcesses))
	heartbeat.Set(1)

	pc := pusher.Client{
		URL:      a.cfg.PushgatewayURL,
		Token:    a.cfg.Token,
		Username: a.cfg.Username,
		Password: a.cfg.Password,
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

func (a *Agent) UpdateInterval() float64 {
	return a.cfg.AutoUpdate.CheckIntervalHours
}
