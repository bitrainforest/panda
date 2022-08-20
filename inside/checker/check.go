package checker

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bitrainforest/PandaAgent/inside/config"
	"github.com/rs/zerolog/log"
)

const (
	AgentStatusNormal = 2
)

type Checker struct {
	sync.Mutex
	cli            *http.Client
	minerID        string
	checkURL       string
	pingURL        string
	checkFrequency time.Duration
	heartFrequency time.Duration
	doneCtx        context.Context
	cancle         context.CancelFunc
	token          string
	// the total sectors this agent need download
	sectorsTotal int64
}

func InitChecker(conf config.Config, parentCtx context.Context) Checker {
	var c Checker
	c.cli = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		Timeout: time.Duration(conf.GH.Timeout) * time.Second,
	}

	ctx, cancle := context.WithCancel(parentCtx)
	c.checkURL = conf.GH.QueryURL
	c.pingURL = conf.GH.PingURL
	c.checkFrequency = conf.GH.CheckFrequency
	c.heartFrequency = conf.GH.HeartFrequency
	c.minerID = conf.Miner.ID
	c.doneCtx = ctx
	c.cancle = cancle
	c.token = conf.GH.Token

	return c
}

func (c Checker) Ping() {
	go func() {
		ticker := time.Tick(c.heartFrequency)
		for {
			select {
			case <-c.doneCtx.Done():
				log.Info().Msgf("[Checker] Heart Stop.")
			case <-ticker:
				log.Info().Msgf("[Checker] do Heart.")
				err := c.ping()
				if err != nil {
					log.Error().Msgf("[Checker] Heart err: %s", err)
					continue
				}
			}
		}
	}()
}

type AgentStatus struct {
	Status       int   `json:"status,omitempty"`
	NeedDownload int64 `json:"need_download,omitempty"`
}

// just ping, we do not hold the connection.
func (c Checker) ping() error {
	as := AgentStatus{
		Status: AgentStatusNormal,
	}
	c.Lock()
	as.NeedDownload = c.sectorsTotal
	c.Unlock()

	content, err := json.Marshal(as)
	if err != nil {
		return err
	}

	log.Debug().Msgf("[Checker] ping content: %+v", as)

	req, err := http.NewRequest("POST", c.pingURL, bytes.NewReader(content))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("minerToken", c.token)
	resp, err := c.cli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("Checker ping err status: %d", resp.StatusCode)
	}

	return nil
}

// checker will get downloadable sectors and send it to channel ch
func (c Checker) Check(ch chan Sector) {
	go func() {
		ticker := time.Tick(c.checkFrequency)
		for {
			select {
			case <-c.doneCtx.Done():
				log.Info().Msgf("[Checker] Check Stop.")
				return
			case <-ticker:
				log.Info().Msgf("[Checker] do Check.")
				res, err := c.check()
				if err != nil {
					log.Error().Msgf("[Checker] Check err: %s", err)
					continue
				}

				if res == nil {
					continue
				}

				c.Lock()
				c.sectorsTotal += int64(len(res))
				c.Unlock()

				if len(res) > 0 {
					for _, v := range res {
						log.Info().Msgf("[Checker] Check get miner: %s sector: %d to download", c.minerID, v)
						ch <- v
					}
				}
			}
		}
	}()
}

type Sector struct {
	ID int
	// we max retry three times
	Try int
}

type checkResponse struct {
	Code int    `json:"code,omitempty"`
	Msg  string `json:"msg,omitempty"`
	Now  int    `json:"nowTime,omitempty"`
	Data Data   `json:"data,omitempty"`
}

type Data struct {
	List []DataItem `json:"list,omitempty"`
}

type DataItem struct {
	MinerID    string `json:"minerId,omitempty"`
	SectorId   string `json:"sectorId,omitempty"`
	SectorType string `json:"sectorType,omitempty"`
}

func (c Checker) check() ([]Sector, error) {
	req, err := http.NewRequest("GET", c.checkURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("minerToken", c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("Checker check err status: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result checkResponse
	if http.StatusNoContent != resp.StatusCode {
		if jsonErr := json.Unmarshal(body, &result); jsonErr != nil {
			return nil, fmt.Errorf("%s: %s", "bad_response", jsonErr.Error())
		}
	}

	if strings.ToLower(result.Msg) != "success" {
		return nil, fmt.Errorf("Checker response msg: %s", result.Msg)
	}

	if len(result.Data.List) > 0 {
		sectors := make([]Sector, 0, 5)
		for _, item := range result.Data.List {
			if item.MinerID != c.minerID {
				continue
			}

			sectorID, _ := strconv.Atoi(item.SectorId)
			sectors = append(sectors, Sector{
				ID:  sectorID,
				Try: 0,
			})
		}

		return sectors, nil
	}

	log.Debug().Msgf("[Checker] there is no sectors need download")

	return nil, nil
}

func (c Checker) Stop() {
	log.Info().Msgf("[Checker] Stop.")
	c.cancle()
}
