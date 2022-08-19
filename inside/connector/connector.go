// Note: this package is not used now, just for websocket in future!
package connector

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bitrainforest/PandaAgent/pkg/util"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

type ConnectStatus int32

const (
	ConnectOpen ConnectStatus = iota
	ConnectClosed
	ConnectReconnect
)

var (
	ErrConnectClosed    = errors.New("connection closed")
	ErrConnectReconnect = errors.New("connection reconnecting")
)

// HConn examines the liveness of the underground connection periodically, and reconnects
// to the remote when the underground connection is unexpectedly closed
type HConn struct {
	dialer *websocket.Dialer
	ws     *websocket.Conn
	status ConnectStatus // atomic
	url    string
	header http.Header

	doneCh    chan struct{}
	closeOnce sync.Once
}

func NewConn(urlStr string, header http.Header) *HConn {
	c := &HConn{
		dialer: websocket.DefaultDialer,
		url:    urlStr,
		header: header.Clone(),
		status: ConnectClosed,

		doneCh: make(chan struct{}),
	}

	go c.health()
	go c.autoConnect()
	return c
}

func (c *HConn) SetWriteDeadline(t time.Time) error {
	if status := getConnStatus(&c.status); status != ConnectOpen {
		c.reconnect()
		return statusErr(status)
	}
	err := c.ws.SetWriteDeadline(t)
	if err != nil {
		log.Error().Err(err).Msg("set write deadline failed")
	}
	return err
}

func (c *HConn) WriteMessage(messageType int, data []byte) error {
	if status := getConnStatus(&c.status); status != ConnectOpen {
		return statusErr(status)
	}
	err := c.ws.WriteMessage(messageType, data)
	if err != nil {
		c.reconnect()
		return fmt.Errorf("write message error: %s", err)
	}
	return nil
}

func (c *HConn) ReadMessage() (int, []byte, error) {
	if status := getConnStatus(&c.status); status != ConnectOpen {
		return -1, nil, statusErr(status)
	}
	t, p, err := c.ws.ReadMessage()
	if err != nil {
		c.reconnect()
		return -1, nil, fmt.Errorf("read message error: %s", err)
	}
	return t, p, err
}

func closeHandler(code int, text string) error {
	log.Info().Int("code", code).Str("text", text).Msg("remote close")
	return nil
}

func (c *HConn) Close() {
	setConnStatus(&c.status, ConnectClosed)
	c.closeOnce.Do(func() {
		close(c.doneCh)
		if c.ws != nil {
			c.ws.Close()
		}
	})
}

func (c *HConn) connect() error {
	// todo: config 配置 cluster, host, uuid
	cluster := ""
	host := ""
	uuid := ""

	timestamp := time.Now().Unix()
	tmp := []string{"AGENT CONNECTION", cluster, host, uuid, strconv.FormatInt(timestamp, 10)}
	text := strings.Join(tmp, "\n")
	sig, err := util.EncryptRSA([]byte(text), "./conf/public.pem")
	if err != nil {
		panic(fmt.Errorf("encrypt error: %s", err))
	}
	sigBase64 := make([]byte, base64.StdEncoding.EncodedLen(len(sig)))
	base64.StdEncoding.Encode(sigBase64, sig)
	header := c.header.Clone()
	header.Add("X-Luna-Signature", string(sigBase64))
	ws, resp, err := c.dialer.Dial(c.url, header)
	if err != nil {
		var errMsg []byte
		var err2 error
		if resp != nil {
			errMsg, err2 = io.ReadAll(resp.Body)
			defer resp.Body.Close()
			if err2 != nil {
				log.Error().Err(err).Msg("websocket dial failed")
				return err2
			}
		}
		log.Error().Err(err).Str("response", string(errMsg)).Msg("websocket dial failed")
		return err
	}
	log.Info().Str("remote conn", ws.RemoteAddr().String()).Msg("websocket connect successfully")

	ws.SetCloseHandler(closeHandler)

	c.ws = ws
	setConnStatus(&c.status, ConnectOpen)
	return nil
}

func (c *HConn) reconnect() {
	if getConnStatus(&c.status) != ConnectOpen {
		return
	}
	setConnStatus(&c.status, ConnectReconnect)
	if c.ws != nil {
		c.ws.Close()
	}
}

func (c *HConn) autoConnect() {
	defer log.Info().Msg("stop auto connect")
	log.Info().Msg("run auto connect")
	tick := time.NewTicker(3 * time.Second)
	for {
		select {
		case <-tick.C:
			if getConnStatus(&c.status) == ConnectOpen {
				continue
			}
			err := c.connect()
			if err != nil {
				continue
			}
		case <-c.doneCh:
			return
		}
	}
}

func (c *HConn) CheckHealth() bool {
	return getConnStatus(&c.status) == ConnectOpen
}

func (c *HConn) health() {
	tick := time.NewTicker(3 * time.Second)
	for {
		select {
		case <-tick.C:
			if getConnStatus(&c.status) != ConnectOpen {
				continue
			}
			err := c.ws.WriteControl(websocket.PingMessage, nil, time.Now().Add(time.Second))
			if err != nil {
				log.Error().Err(err).Msg("websocket ping failed")
				c.reconnect()
				continue
			}
		case <-c.doneCh:
			return
		}
	}
}

func setConnStatus(s *ConnectStatus, val ConnectStatus) {
	atomic.StoreInt32((*int32)(s), int32(val))
}

func getConnStatus(s *ConnectStatus) ConnectStatus {
	ret := atomic.LoadInt32((*int32)(s))
	return ConnectStatus(ret)
}

func (c *HConn) Status() ConnectStatus {
	return getConnStatus(&c.status)
}

func statusErr(status ConnectStatus) error {
	switch status {
	case ConnectOpen:
		return nil
	case ConnectClosed:
		return ErrConnectClosed
	case ConnectReconnect:
		return ErrConnectReconnect
	}
	return nil
}
