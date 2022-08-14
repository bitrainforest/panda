package engine

import (
	"context"

	"github.com/pandarua-agent/inside/checker"
	"github.com/pandarua-agent/inside/config"
	"github.com/pandarua-agent/inside/downloader"
	"github.com/rs/zerolog/log"
)

type Engine struct {
	Transformer downloader.Transformer
	Checker     checker.Checker
	Buf         chan checker.Sector
	ctx         context.Context
	cancle      context.CancelFunc
}

func InitEngine(conf config.Config, ctx context.Context) Engine {
	var engine Engine
	engine.Transformer = downloader.InitTransformer(conf, ctx)
	engine.Checker = checker.InitChecker(conf, ctx)
	engine.Buf = make(chan checker.Sector, 1024)
	engine.ctx, engine.cancle = context.WithCancel(ctx)
	return engine
}

func (eg Engine) Run() error {
	log.Info().Msgf("[Engine] Engine Start.")
	eg.Checker.Ping()
	//eg.Checker.Check(eg.Buf)
	// todo: test code
	eg.Buf <- checker.Sector{ID: 21}
	eg.Transformer.Run(eg.Buf)
	return nil
}

func (eg Engine) Stop() {
	log.Info().Msgf("[Engine] Engine Stop.")
	eg.cancle()
}
