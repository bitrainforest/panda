package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/imdario/mergo"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"
)

const (
	ENV_PANDA_PLATFORM_DOWNLOAD_QUERY    = "ENV_PANDA_PLATFORM_DOWNLOAD_QUERY"
	ENV_PANDA_PLATFORM_DOWNLOAD_CALLBACK = "ENV_PANDA_PLATFORM_DOWNLOAD_CALLBACK"
	ENV_PANDA_PLATFORM_DOWNLOAD          = "ENV_PANDA_PLATFORM_DOWNLOAD"
	ENV_PANDA_PLATFORM_HEART             = "ENV_PANDA_PLATFORM_HEART"
	ENV_PANDA_LOGDIR_DEFAULT             = "ENV_PANDA_LOGDIR_DEFAULT"
)

// global app config variable
var (
	AppConfig Config
)

type Config struct {
	ConfigDir   string `yaml:"-"`
	Env         string `yaml:"-"`
	Transformer struct {
		MaxDownloader            int    `yaml:"MaxParallelNumber"`
		MaxDownloadRetry         int    `yaml:"MaxRetryNumber"`
		TransformPartSize        int    `yaml:"SliceSize"`
		SingleDownloadMaxWorkers int    `yaml:"MaxSliceNumber"`
		WorkDir                  string `yaml:"WorkDir"`
	} `yaml:"Transmission"`
	Miner struct {
		SealedPath      string `yaml:"StoreSealedPath"`
		SealedCachePath string `yaml:"StoreCachePath"`
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

	if AppConfig.GH.QueryURL == "" {
		AppConfig.GH.QueryURL = os.Getenv(ENV_PANDA_PLATFORM_DOWNLOAD_QUERY)
	}

	if AppConfig.GH.CallBack == "" {
		AppConfig.GH.CallBack = os.Getenv(ENV_PANDA_PLATFORM_DOWNLOAD_CALLBACK)
	}

	if AppConfig.GH.DownloadURL == "" {
		AppConfig.GH.DownloadURL = os.Getenv(ENV_PANDA_PLATFORM_DOWNLOAD)
	}

	if AppConfig.GH.PingURL == "" {
		AppConfig.GH.PingURL = os.Getenv(ENV_PANDA_PLATFORM_HEART)
	}

	if AppConfig.Log.Dir == "" {
		AppConfig.Log.Dir = os.Getenv(ENV_PANDA_LOGDIR_DEFAULT)
	}

	return nil
}

func parseYAMLFile(filePath string) error {
	if AppConfig.Env == "" {
		AppConfig.Env = "Default"
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
