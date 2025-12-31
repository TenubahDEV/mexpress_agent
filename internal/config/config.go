package config

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

type AutoUpdateConfig struct {
	Enabled            bool `yaml:"enabled"`
	CheckIntervalHours float64 `yaml:"check_interval_hours"`
}

type Config struct {
	JobName         string            `yaml:"job_name"`
	InstanceName    string            `yaml:"instance_name"`
	PushgatewayURL  string            `yaml:"pushgateway_url"`
	Token           string            `yaml:"token"`
	IntervalSeconds int               `yaml:"interval_seconds"`
	Labels          map[string]string `yaml:"labels"`
	AutoUpdate      AutoUpdateConfig  `yaml:"auto_update"`
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
	if c.Token == "" {
		return nil, errors.New("token required")
	}
	if c.AutoUpdate.CheckIntervalHours <= 0 {
		c.AutoUpdate.CheckIntervalHours = 24
	}

	return &c, nil
}
