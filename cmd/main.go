package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"

	"github.com/pandarua-agent/inside/config"
	"github.com/urfave/cli/v2"
)

const (
	NAME = "github.com/pandarua-agent"
)

func init() {
	cli.VersionPrinter = printVersion
}

var (
	buildstamp string
	githash    string
	version    = "No version"
)

func printVersion(ctx *cli.Context) {
	fmt.Println("Version    :", version)
	fmt.Println("Go Version :", runtime.Version())
	fmt.Println("Git Commit :", githash)
	fmt.Println("Build Time :", buildstamp)
}

func main() {
	go func() {
		log.Println(http.ListenAndServe(":6060", nil))
	}()

	app := &cli.App{
		Name:    NAME,
		Version: version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "env",
				Value:       "",
				EnvVars:     []string{"ENV"},
				Usage:       "enviroment, can be Develop, Product, or some other cumtomize value",
				Destination: &config.AppConfig.ConfigDir,
			},
			&cli.StringFlag{
				Name:        "conf-dir",
				Value:       "",
				EnvVars:     []string{"CONFDIR"},
				Usage:       "configuration file directory",
				Destination: &config.AppConfig.ConfigDir,
			},
			&cli.StringFlag{
				Name:        "log-dir",
				Value:       "",
				EnvVars:     []string{"LOGDIR"},
				Usage:       "the log dir",
				Destination: &config.AppConfig.LogDir,
			},
		},
		Commands: []*cli.Command{
			{
				Name:   "run",
				Usage:  "run service",
				Action: run,
			},
		},
	}

	app.Run(os.Args)
}
