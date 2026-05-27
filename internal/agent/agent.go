package agent

import (
	"context"
	"log"
	"os"
	"time"

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
	info := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tenubah_agent_info",
		Help: "Information about the agent and host system",
		ConstLabels: prometheus.Labels{
			"ip":               m.IP,
			"os":               m.OS,
			"platform":         m.Platform,
			"platform_version": m.PlatformVersion,
			"version":          version.Version,
		},
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
		info,
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
	info.Set(1)

	pc := pusher.Client{
		URL:      a.cfg.PushgatewayURL,
		Token:    a.cfg.Token,
		Username: a.cfg.Username,
		Password: a.cfg.Password,
	}

	// Copiar etiquetas globales y añadir component: system para evitar colisión en Pushgateway
	labels := make(map[string]string)
	for k, v := range a.cfg.Labels {
		labels[k] = v
	}
	labels["component"] = "system"

	// 🔥 Push usando Gatherer
	return pc.PushGatherer(
		a.cfg.JobName,
		a.instance(),
		labels,
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

func (a *Agent) DatabaseMonitoringEnabled() bool {
	return a.cfg.DatabaseMonitoring.Enabled
}

func (a *Agent) StartDatabaseMonitoring(quit chan struct{}) {
	interval := time.Duration(a.cfg.DatabaseMonitoring.CollectIntervalSeconds) * time.Second
	log.Printf("Starting database monitoring loop, type=%s, interval=%v", a.cfg.DatabaseMonitoring.Type, interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Ejecutar inmediatamente al inicio
	a.runDatabaseMonitoringOnce()

	for {
		select {
		case <-quit:
			collectors.Close() // cerrar conexiones abiertas al salir
			return
		case <-ticker.C:
			a.runDatabaseMonitoringOnce()
		}
	}
}

func (a *Agent) runDatabaseMonitoringOnce() {
	// Crear contexto con timeout de 10s para evitar bloqueos
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	m, err := collectors.CollectSQLServer(ctx, a.cfg.DatabaseMonitoring.ConnectionString)
	if err != nil {
		log.Printf("WARNING sqlserver: SQL Server collection returned errors: %v", err)
	}

	reg := prometheus.NewRegistry()

	// 1. Métricas generales del motor
	up := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tenubah_sqlserver_up",
		Help: "SQL Server status (1 = UP, 0 = DOWN)",
	})
	userConns := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tenubah_sqlserver_user_connections",
		Help: "Active SQL Server user connections",
	})
	dbCount := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tenubah_sqlserver_database_count",
		Help: "Total databases in SQL Server",
	})
	failedJobs := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tenubah_sqlserver_failed_jobs_last_24h",
		Help: "Failed SQL Agent jobs in the last 24 hours",
	})

	reg.MustRegister(up, userConns, dbCount, failedJobs)

	up.Set(m.Up)
	userConns.Set(m.UserConnections)
	dbCount.Set(m.DatabaseCount)
	failedJobs.Set(m.FailedJobsLast24h)

	// 2. Métricas por base de datos (con etiqueta 'database')
	dbStatus := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tenubah_sqlserver_database_status",
		Help: "Database status (1 = ONLINE, 0 = NOT ONLINE)",
	}, []string{"database"})

	dbSize := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tenubah_sqlserver_database_size_mb",
		Help: "Database size in MB",
	}, []string{"database"})

	dbLogSize := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tenubah_sqlserver_log_size_mb",
		Help: "Log file size in MB",
	}, []string{"database"})

	dbLogUsed := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tenubah_sqlserver_log_used_percent",
		Help: "Percentage of log file used",
	}, []string{"database"})

	dbFullBackup := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tenubah_sqlserver_last_full_backup_age_hours",
		Help: "Hours since last successful full backup (-1 if never)",
	}, []string{"database"})

	dbDiffBackup := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tenubah_sqlserver_last_diff_backup_age_hours",
		Help: "Hours since last successful differential backup (-1 if never)",
	}, []string{"database"})

	dbLogBackup := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tenubah_sqlserver_last_log_backup_age_minutes",
		Help: "Minutes since last successful transaction log backup (-1 if never)",
	}, []string{"database"})

	reg.MustRegister(dbStatus, dbSize, dbLogSize, dbLogUsed, dbFullBackup, dbDiffBackup, dbLogBackup)

	// Cargar dinámicamente las bases de datos registradas
	for _, db := range m.Databases {
		dbStatus.WithLabelValues(db.Name).Set(db.Status)
		dbSize.WithLabelValues(db.Name).Set(db.SizeMB)
		dbLogSize.WithLabelValues(db.Name).Set(db.LogSizeMB)
		dbLogUsed.WithLabelValues(db.Name).Set(db.LogUsedPercent)
		dbFullBackup.WithLabelValues(db.Name).Set(db.LastFullBackupAgeHours)
		dbDiffBackup.WithLabelValues(db.Name).Set(db.LastDiffBackupAgeHours)
		dbLogBackup.WithLabelValues(db.Name).Set(db.LastLogBackupAgeMinutes)
	}

	// Cliente de push hacia Pushgateway
	pc := pusher.Client{
		URL:      a.cfg.PushgatewayURL,
		Token:    a.cfg.Token,
		Username: a.cfg.Username,
		Password: a.cfg.Password,
	}

	labels := make(map[string]string)
	for k, v := range a.cfg.Labels {
		labels[k] = v
	}
	labels["component"] = "database"

	if err := pc.PushGatherer(a.cfg.JobName, a.instance(), labels, reg); err != nil {
		log.Printf("ERROR sqlserver: Failed to push SQL Server metrics to Pushgateway: %v", err)
	} else {
		log.Printf("Successfully pushed SQL Server metrics to Pushgateway")
	}
}
