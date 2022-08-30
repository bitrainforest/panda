package deal

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/bitrainforest/PandaAgent/inside/boost"
	"github.com/bitrainforest/PandaAgent/inside/config"
	"github.com/rs/zerolog/log"
)

type DealTransform struct {
	dealTransformURL string
	cli              *http.Client
	frequency        time.Duration
	doneCtx          context.Context
	cancle           context.CancelFunc
	token            string
	boostCli         *boost.BoostCli
	ch               chan *boost.Deal
	buffer           []*boost.Deal
	maxBuffer        int
}

func InitDealTransform(conf config.Config, parentCtx context.Context) *DealTransform {
	var dt DealTransform
	dt.cli = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			MaxIdleConns:          50,
			MaxIdleConnsPerHost:   1,
			MaxConnsPerHost:       10,
			IdleConnTimeout:       10 * time.Second,
		},
		Timeout: time.Duration(conf.GH.Timeout) * time.Second,
	}

	ctx, cancle := context.WithCancel(parentCtx)
	dt.cancle = cancle
	dt.doneCtx = ctx
	dt.token = conf.GH.Token
	dt.frequency = conf.GH.DealFrequency
	dt.dealTransformURL = conf.GH.DealURL
	// 10 need configurable
	dt.ch = make(chan *boost.Deal, 10)
	dt.boostCli = boost.InitBoostCli(conf.Boost.Address, conf.Boost.GraphQlURL, conf.Boost.APIToken, dt.ch)
	dt.buffer = make([]*boost.Deal, 0, 10)
	dt.maxBuffer = 10

	return &dt
}

func (dt *DealTransform) Run() {
	go dt.boostCli.Start()
	go func() {
		ticker := time.Tick(dt.frequency)
		for {
			select {
			case <-dt.doneCtx.Done():
				log.Info().Msgf("[DealTransform] Stop.")
				return
			case <-ticker:
				if len(dt.buffer) > 0 {
					log.Info().Msgf("[DealTransform] do Transform.")
					dt.Transform()
					dt.buffer = dt.buffer[:0]
				}
			case d, ok := <-dt.ch:
				if !ok {
					log.Warn().Msgf("[DealTransform] channel closed.")
					return
				}
				log.Debug().Msgf("[DealTransform] receive deal: %s", d.UUID)
				if len(dt.buffer) >= dt.maxBuffer {
					log.Info().Msgf("[DealTransform] do Transform as buffer is full.")
					dt.Transform()
					dt.buffer = dt.buffer[:0]
				}

				dt.buffer = append(dt.buffer, d)
			}
		}
	}()
}

type DealContent struct {
	Total   int          `json:"total,omitempty"`
	Extra   string       `json:"extra,omitempty"`
	Content []boost.Deal `json:"content,omitempty"`
}

func (dt *DealTransform) Transform() error {
	var c DealContent
	c.Content = make([]boost.Deal, 0, len(dt.buffer))
	c.Total = len(dt.buffer)
	for _, d := range dt.buffer {
		c.Content = append(c.Content, *d)
	}

	content, err := json.Marshal(&c)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", dt.dealTransformURL, bytes.NewReader(content))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("minerToken", dt.token)

	resp, err := dt.cli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("DealTransform post err status: %d", resp.StatusCode)
	}

	return nil
}
