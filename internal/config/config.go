package config

import (
	"errors"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type AutoUpdateConfig struct {
	Enabled            bool    `yaml:"enabled"`
	CheckIntervalHours float64 `yaml:"check_interval_hours"`
}

type DatabaseMonitoringConfig struct {
	Enabled                bool   `yaml:"enabled"`
	Type                   string `yaml:"type"`
	ConnectionString       string `yaml:"connection_string"`
	CollectIntervalSeconds int    `yaml:"collect_interval_seconds"`
}

type Config struct {
	JobName            string                   `yaml:"job_name"`
	InstanceName       string                   `yaml:"instance_name"`
	PushgatewayURL     string                   `yaml:"pushgateway_url"`
	Token              string                   `yaml:"token"`
	Username           string                   `yaml:"username"`
	Password           string                   `yaml:"password"`
	IntervalSeconds    int                      `yaml:"interval_seconds"`
	Labels             map[string]string        `yaml:"labels"`
	AutoUpdate         AutoUpdateConfig         `yaml:"auto_update"`
	DatabaseMonitoring DatabaseMonitoringConfig `yaml:"database_monitoring"`
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

	// ENV overrides
	if v := os.Getenv("TENUBAH_TOKEN"); v != "" {
		c.Token = v
	}
	if v := os.Getenv("TENUBAH_PUSH_URL"); v != "" {
		c.PushgatewayURL = v
	}
	if v := os.Getenv("TENUBAH_USER"); v != "" {
		c.Username = v
	}
	if v := os.Getenv("TENUBAH_PASSWORD"); v != "" {
		c.Password = v
	}
	if v := os.Getenv("TENUBAH_DB_CONN"); v != "" {
		c.DatabaseMonitoring.ConnectionString = v
	}
	if v := os.Getenv("TENUBAH_DB_TYPE"); v != "" {
		c.DatabaseMonitoring.Type = v
	}

	// Limpieza robusta del token
	// 1. Quitar espacios extremos
	// 2. Quitar prefijo "Bearer" (case sensitive, requiere espacio exacto o se ajusta)
	// 3. Quitar espacios sobrantes de nuevo
	t := strings.TrimSpace(c.Token)
	t = strings.TrimPrefix(t, "Bearer ")
	c.Token = strings.TrimSpace(t)

	c.Username = strings.TrimSpace(c.Username)
	c.Password = strings.TrimSpace(c.Password)
	c.PushgatewayURL = strings.TrimSpace(c.PushgatewayURL)

	if c.IntervalSeconds <= 0 {
		c.IntervalSeconds = 60
	}
	if c.Labels == nil {
		c.Labels = map[string]string{}
	}

	if c.JobName == "" {
		return nil, errors.New("job_name required")
	}
	if c.PushgatewayURL == "" {
		return nil, errors.New("pushgateway_url required")
	}
	if c.Token == "" && (c.Username == "" || c.Password == "") {
		return nil, errors.New("either token or username/password required")
	}
	if c.AutoUpdate.CheckIntervalHours <= 0 {
		c.AutoUpdate.CheckIntervalHours = 24
	}

	if c.DatabaseMonitoring.Enabled {
		c.DatabaseMonitoring.Type = strings.TrimSpace(strings.ToLower(c.DatabaseMonitoring.Type))
		c.DatabaseMonitoring.ConnectionString = strings.TrimSpace(c.DatabaseMonitoring.ConnectionString)

		if c.DatabaseMonitoring.Type == "" {
			return nil, errors.New("database_monitoring.type is required when enabled")
		}
		if c.DatabaseMonitoring.Type != "sqlserver" {
			return nil, errors.New("database_monitoring.type must be 'sqlserver'")
		}
		if c.DatabaseMonitoring.ConnectionString == "" {
			return nil, errors.New("database_monitoring.connection_string is required when enabled")
		}
		if c.DatabaseMonitoring.CollectIntervalSeconds <= 0 {
			c.DatabaseMonitoring.CollectIntervalSeconds = 60
		}
	}

	return &c, nil
}
