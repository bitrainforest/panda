package config

import (
	"fmt"
	"io/ioutil"
	"path"
	"strings"
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
	LogDir      string `yaml:"LogDir"`
	ConfigDir   string `yaml:"-"`
	Env         string `yaml:"-"`
	Transformer struct {
		MaxDownloader            int `yaml:"MaxDownloader"`
		MaxDownloadRetry         int `yaml:"MaxDownloadRetry"`
		TransformPartSize        int `yaml:"TransformPartSize"`
		SingleDownloadMaxWorkers int `yaml:"SingleDownloadMaxWorkers"`
	} `yaml:"Transformer"`
	Miner struct {
		SealedPath      string `yaml:"MinerSealedPath"`
		SealedCachePath string `yaml:"MinerSealedCachePath"`
		APIToken        string `yaml:"APIToken"`
		ID              string `yaml:"ID"`
		StorageID       string `yaml:"StorageID"`
	} `yaml:"Miner"`
	Log struct {
		Level string `yaml:"Level"`
		Dir   string `yaml:"Dir"`
	} `yaml:"Log"`
	GH struct {
		Address        string        `yaml:"Address"`
		QueryURL       string        `yaml:"QueryURL"`
		CallBack       string        `yaml:"CallBack"`
		DownloadURL    string        `yaml:"DownloadURL"`
		Timeout        int           `yaml:"timeout"`
		PingURL        string        `yaml:"HeartURL"`
		CheckFrequency time.Duration `yaml:"CheckFrequency"`
		HeartFrequency time.Duration `yaml:"HeartFrequency"`
		Token          string        `yaml:"token"`
	} `yaml:"GH"`
}

// Init parse the yaml configuration file
func Init(ctx *cli.Context) error {
	if AppConfig.ConfigDir != "" {
		if err := parseYAMLFile(path.Join(AppConfig.ConfigDir, strings.Replace(ctx.String("psm"), ".", "_", -1)+".yaml")); err != nil {
			panic(err)
		}
	}

	return nil
}

func parseYAMLFile(filePath string) error {
	// todo: make config adjust the env
	AppConfig.Env = "Product"

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
