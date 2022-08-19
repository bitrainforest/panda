package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path"
	"strconv"
	"syscall"

	"github.com/bitrainforest/PandaAgent/inside/config"
	"github.com/bitrainforest/PandaAgent/inside/engine"
	logwriter "github.com/bitrainforest/PandaAgent/inside/log"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

func init() {
	zerolog.CallerMarshalFunc = func(file string, line int) string {
		return path.Base(file) + ":" + strconv.Itoa(line)
	}
	cw := logwriter.New()
	cw.NoColor = true
	cw.Out = os.Stdout
	cw.TimeFormat = "3:04:05PM"
	log.Logger = zerolog.New(cw).With().Caller().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

func initConfig(ctx *cli.Context) {
	if err := config.Init(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to parse configuration file")
	}

	return
}

func run(ctx *cli.Context) error {
	initConfig(ctx)

	logLevelString := os.Getenv("LOG_LEVEL")
	if logLevelString == "" {
		logLevelString = config.AppConfig.Log.Level
	}

	if logLevelString == "" {
		logLevelString = "debug"
	}

	f, err := os.OpenFile(config.GetConfig().Log.Dir, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		fmt.Println(err)
	} else {
		defer f.Close()
		log.Logger = log.Output(f)
	}

	lvl, err := zerolog.ParseLevel(logLevelString)
	if err == nil {
		zerolog.SetGlobalLevel(lvl)
	} else {
		log.Warn().Str("value", logLevelString).Err(err).Msg("Invalid level value, using info by default")
	}

	log.Info().Interface("conf", config.GetConfig()).Msg("Print the Config")
	log.Info().Msg("starting the agent...")
	eg := engine.InitEngine(config.GetConfig(), context.Background())

	if err := eg.Run(); err != nil {
		log.Fatal().Err(err).Msg("Failed to run engine")
	}

	defer eg.Stop()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)

	for {
		select {
		case sig := <-ch:
			fmt.Printf("Got signal: %s, Exit..\n", sig)
			return errors.New(sig.String())
		}
	}

	return nil
}
