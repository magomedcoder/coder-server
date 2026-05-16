package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type GenerateConfig struct {
	MaxTokens   int     `yaml:"max_tokens"`
	Temperature float64 `yaml:"temperature"`
}

type ChatConfig struct {
	TimeoutSeconds int            `yaml:"timeout_seconds"`
	Generate       GenerateConfig `yaml:"generate"`
}

type AgentConfig struct {
	MaxTokens   int     `yaml:"max_tokens"`
	Temperature float64 `yaml:"temperature"`
}

type Config struct {
	Host       string      `yaml:"host"`
	Port       int         `yaml:"port"`
	RunnerAddr string      `yaml:"runner_addr"`
	Chat       ChatConfig  `yaml:"chat"`
	Agent      AgentConfig `yaml:"agent"`

	listenOverride string
}

func Load(path string) (*Config, error) {
	c := &Config{}

	if _, err := os.Stat(path); err == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("ошибка чтения конфигурационного файла %s: %w", path, err)
		}

		if err := yaml.Unmarshal(data, c); err != nil {
			return nil, fmt.Errorf("ошибка парсинга конфигурационного файла %s: %w", path, err)
		}
	}

	return c, nil
}

func (c *Config) ChatTimeoutSeconds() int32 {
	if c == nil || c.Chat.TimeoutSeconds <= 0 {
		return 300
	}
	return int32(c.Chat.TimeoutSeconds)
}

func (c *Config) ListenAddr() string {
	if c.listenOverride != "" {
		return c.listenOverride
	}

	return net.JoinHostPort(strings.TrimSpace(c.Host), strconv.Itoa(c.Port))
}
