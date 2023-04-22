package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Debug bool `yaml:"debug"`
	Port  int  `yaml:"port"`
	IPV6  bool `yaml:"ipv6"`

	VipOnly bool `yaml:"vipOnly"`

	BlacklistApiUrl string        `yaml:"blacklistApiUrl"`
	BlockType       BlockTypeEnum `yaml:"blockType"`

	RoamingMinVer int `yaml:"roamingMinVer"`

	DefaultArea string `yaml:"defaultArea"`

	ThRedirect struct {
		Aid int `yaml:"aid"`
	} `yaml:"thRedirect"`

	Limiter struct {
		Limit int `yaml:"limit"`
		Burst int `yaml:"burst"`
	} `yaml:"limiter"`

	SearchLimiter struct {
		Limit int `yaml:"limit"`
		Burst int `yaml:"burst"`
	} `yaml:"searchLimiter"`

	CustomSearch struct {
		Data    string `yaml:"data"`
		WebData string `yaml:"webData"`
	} `yaml:"customSearch"`

	CustomSubtitle struct {
		ApiUrl   string `yaml:"apiUrl"`
		TeamName string `yaml:"teamName"`
	} `yaml:"customSubtitle"`

	Cache struct {
		AccessKey  time.Duration `yaml:"accessKey"`
		User       time.Duration `yaml:"user"`
		PlayUrl    time.Duration `yaml:"playUrl"`
		THSeason   time.Duration `yaml:"thSeason"`
		THSubtitle time.Duration `yaml:"thSubtitle"`
	} `yaml:"cache"`

	Proxy struct {
		CN      string `yaml:"cn"`
		HK      string `yaml:"hk"`
		TW      string `yaml:"tw"`
		TH      string `yaml:"th"`
		Default string `yaml:"default"`
	} `yaml:"proxy"`

	Reverse struct {
		CN string `yaml:"cn"`
		HK string `yaml:"hk"`
		TW string `yaml:"tw"`
		TH string `yaml:"th"`
	} `yaml:"reverse"`

	ReverseSearch struct {
		CN string `yaml:"cn"`
		HK string `yaml:"hk"`
		TW string `yaml:"tw"`
		TH string `yaml:"th"`
	} `yaml:"reverseSearch"`

	ReverseWebSearch struct {
		CN string `yaml:"cn"`
		HK string `yaml:"hk"`
		TW string `yaml:"tw"`
	} `yaml:"reverseWebSearch"`

	Auth struct {
		CN bool `yaml:"cn"`
		HK bool `yaml:"hk"`
		TW bool `yaml:"tw"`
		TH bool `yaml:"th"`
	} `yaml:"auth"`

	PostgreSQL struct {
		Host         string `yaml:"host"`
		User         string `yaml:"user"`
		Password     string `yaml:"password"`
		PasswordFile string `yaml:"passwordFile"`
		DBName       string `yaml:"dbName"`
		Port         int    `yaml:"port"`
	} `yaml:"postgreSQL"`
}

func validateConfigPath(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		return fmt.Errorf("'%s' is a directory", path)
	}
	return nil
}

func parseFlags() (string, error) {
	var configPath string

	flag.StringVar(&configPath, "config", "./config.yml", "Path to config file")

	flag.Parse()

	if err := validateConfigPath(configPath); err != nil {
		return "", err
	}

	return configPath, nil
}

func initConfig(configPath string) (*Config, error) {
	config := &Config{}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return config, nil
}

func (c *Config) saveConfig(configPath string) error {
	data, err := yaml.Marshal(&c)
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return err
	}

	return nil
}
