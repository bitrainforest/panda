// Note: this package is not used now, just for websocket in future!
package client

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	// todo: fix the import path named internal replace inside
	"github.com/pandarua-agent/inside/connector"
	"github.com/pandarua-agent/pkg/util"
	"github.com/rs/zerolog/log"
)

var (
	c          *client
	numWorkers = 8
)

type client struct {
	ws *connector.HConn

	readCh  chan *connector.Message
	writeCh chan *connector.Message

	//workers  []*downloader
	workerCh chan *connector.Message

	doneCh chan struct{}

	retryMax int64
}

func Init() {
	//todo: config
	/*
		cfg := config.Get()
		domain := cfg.Server.Domain
		path := cfg.Server.Path
		u, err := url.Parse(domain)
		if err != nil {
			panic(fmt.Errorf("domain error: %s", err))
		}
		u.Path = path

		cluster := cfg.Cluster
		host := cfg.HostIP
		uuid := cfg.UUID


	*/
	header := http.Header{}
	// tlb需要host头
	c = &client{
		ws:      connector.NewConn("localhost:1234/websocket", header),
		readCh:  make(chan *connector.Message, 16),
		writeCh: make(chan *connector.Message, 16),
		// todo: downloader implement
		//workers:  make([]*downloader, 0),
		workerCh: make(chan *connector.Message, 16),
		doneCh:   make(chan struct{}),
		retryMax: 5,
	}

	for i := 0; i < numWorkers; i++ {
		//todo: start downloader here
	}

	go c.writePump()
	go c.readPump()
	go c.dispatch()
}

func (c *client) waitAndRetry(counter *int64) {
	wait := util.Pow2(*counter)
	time.Sleep(time.Duration(wait) * time.Second)
	if *counter < c.retryMax {
		(*counter)++
	}
}

func (c *client) resetCounter(counter *int64) {
	*counter = 0
}

func (c *client) writePump() {
	defer log.Info().Msg("write pump stop")
	var writeRetryCnt int64 = 0
	for {
		select {
		case msg := <-c.writeCh:
			s, err := msg.Serialize()
			if err != nil {
				log.Error().Err(err).Msg("message serialization failed")
				continue
			}
		WRITE:
			c.ws.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err = c.ws.WriteMessage(websocket.BinaryMessage, s); err != nil {
				log.Error().Err(err).Msg("websocket write failed")
				c.waitAndRetry(&writeRetryCnt)
				goto WRITE
			} else {
				c.resetCounter(&writeRetryCnt)
			}
		case <-c.doneCh:
			return
		}
	}
}

func (c *client) readPump() {
	defer log.Info().Msg("read pump stop")
	var readRetryCnt int64 = 0
	for {
		if !c.ws.CheckHealth() {
			c.waitAndRetry(&readRetryCnt)
		} else {
			c.resetCounter(&readRetryCnt)
		}

		_, s, err := c.ws.ReadMessage()
		if err != nil {
			log.Error().Err(err).Msg("websocket read failed")
			continue
		}

		// todo: 消息格式还需要定下
		msg, err := connector.MessageBytes(s).Deserialize()
		if err != nil {
			log.Error().Err(err).Msg("message deserialization failed")
			continue
		}
		select {
		case c.readCh <- &msg:
		case <-c.doneCh:
			return
		}
	}
}

func (c *client) dispatch() {
	defer log.Info().Msg("dispatch stop")
	log.Info().Msg("start dispatch")
	for {
		select {
		case msg := <-c.readCh:
			c.workerCh <- msg
		case <-c.doneCh:
			return
		}
	}
}

func Stop() {
	close(c.doneCh)
	c.ws.Close()
}

func (c *client) response(session string, msg *connector.Message) {
	msg.Session = session
	select {
	case c.writeCh <- msg:
		// default:
		// 	log.Error().Err(fmt.Errorf("write channel full")).Msg("client write error")
	}
}
