package config

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/imdario/mergo"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"
)

// global app config variable
var (
	AppConfig Config
)

type Config struct {
	ConfigDir   string `yaml:"-"`
	Env         string `yaml:"-"`
	Transformer struct {
		MaxDownloader            int    `yaml:"MaxDownloader"`
		MaxDownloadRetry         int    `yaml:"MaxDownloadRetry"`
		TransformPartSize        int    `yaml:"TransformPartSize"`
		SingleDownloadMaxWorkers int    `yaml:"SingleDownloadMaxWorkers"`
		WorkDir                  string `yaml:"WorkDir"`
	} `yaml:"Transformer"`
	Miner struct {
		SealedPath      string `yaml:"MinerSealedPath"`
		SealedCachePath string `yaml:"MinerSealedCachePath"`
		APIToken        string `yaml:"APIToken"`
		ID              string `yaml:"ID"`
		StorageID       string `yaml:"StorageID"`
		Address         string `yaml:"Address"`
	} `yaml:"Miner"`
	Log struct {
		Level string `yaml:"Level"`
		Dir   string `yaml:"Dir"`
	} `yaml:"Log"`
	GH struct {
		QueryURL       string        `yaml:"QueryURL"`
		CallBack       string        `yaml:"CallBack"`
		DownloadURL    string        `yaml:"DownloadURL"`
		Timeout        int           `yaml:"Timeout"`
		PingURL        string        `yaml:"HeartURL"`
		CheckFrequency time.Duration `yaml:"CheckFrequency"`
		HeartFrequency time.Duration `yaml:"HeartFrequency"`
		Token          string        `yaml:"Token"`
	} `yaml:"Platform"`
}

// Init parse the yaml configuration file
func Init(ctx *cli.Context) error {
	if AppConfig.ConfigDir != "" {
		if err := parseYAMLFile(AppConfig.ConfigDir); err != nil {
			panic(err)
		}
	}

	return nil
}

func parseYAMLFile(filePath string) error {
	if AppConfig.Env == "" {
		AppConfig.Env = "Testing"
	}

	var data map[string]Config
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(content, &data); err != nil {
		return err
	}

	if conf, ok := data[AppConfig.Env]; ok {
		if err := mergo.Merge(&AppConfig, conf); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("no %s env defined in configuration file", AppConfig.Env)
	}
	return nil
}

// GetConfig return the config
func GetConfig() Config {
	return AppConfig
}
