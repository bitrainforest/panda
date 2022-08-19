package downloader

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bitrainforest/PandaAgent/inside/checker"
	"github.com/bitrainforest/PandaAgent/inside/config"
	"github.com/bitrainforest/PandaAgent/inside/minerclient"
	"github.com/rs/zerolog/log"
)

const (
	ActionDownload           = "download"
	StatusDownloadSuccessful = "success"
	StatusDownloadFailed     = "failed"
	ActionDeclare            = "declare"
	StatusDeclareSuccessful  = "success"
	StatusDeclareFailed      = "failed"
	StatusCodeOK             = 10000
	StatusCodeFailed         = 20000
)

var (
	ErrRetryExceed = errors.New("retry exceed")
)

type Transformer struct {
	cli                      *http.Client
	minerCli                 minerclient.MinerCli
	CacheDir                 string
	SealedDir                string
	minerID                  string
	downloadURL              string
	MaxDownloader            int
	MaxDownloadRetry         int
	singleDownloadMaxWorkers int
	transformPartSize        int
	// ch is used for control the number of downloader
	ch          chan checker.Sector
	ctx         context.Context
	cancel      context.CancelFunc
	callBackURL string
	token       string
	workDir     string
}

func InitTransformer(conf config.Config, ctx context.Context) Transformer {
	t := Transformer{
		cli: &http.Client{
			Timeout: time.Duration(conf.GH.Timeout) * time.Second,
		},
		minerCli:                 minerclient.InitMinerCli(conf),
		CacheDir:                 conf.Miner.SealedCachePath,
		SealedDir:                conf.Miner.SealedPath,
		MaxDownloader:            conf.Transformer.MaxDownloader,
		MaxDownloadRetry:         conf.Transformer.MaxDownloadRetry,
		transformPartSize:        conf.Transformer.TransformPartSize,
		singleDownloadMaxWorkers: conf.Transformer.SingleDownloadMaxWorkers,
		callBackURL:              conf.GH.CallBack,
		minerID:                  conf.Miner.ID,
		downloadURL:              conf.GH.DownloadURL,
		token:                    conf.GH.Token,
		workDir:                  conf.Transformer.WorkDir,
	}
	t.ch = make(chan checker.Sector, t.MaxDownloader)
	t.ctx, t.cancel = context.WithCancel(ctx)

	log.Info().Msgf("[Transformer] init: %+v", t)

	return t
}

// todo: run code need improve
func (t Transformer) Run(buf chan checker.Sector) {
	go func() {
		for {
			select {
			case s, ok := <-buf:
				if !ok {
					log.Warn().Msgf("[Transformer] channel is cloesed, exit")
					break
				}

				log.Debug().Msgf("[Transformer] try download s: %+v", s)
				t.ch <- s
			case <-t.ctx.Done():
				return
			}
		}
	}()

	go func() {
		for {
			select {
			case s, ok := <-t.ch:
				if !ok {
					log.Warn().Msgf("[Transformer] channel is cloesed, exit")
					break
				}

				log.Debug().Msgf("[Transformer] start download s: %+v", s)
				s.Try += 1
				if s.Try > t.MaxDownloadRetry {
					log.Info().Msgf("[Transformer] miner: %s, sector: %d retry too much, do failed callback", t.minerID, s.ID)
					if err := t.CallBack(DownloadCallBackContent{
						Action:     ActionDownload,
						Status:     StatusDownloadFailed,
						StatusCode: StatusCodeFailed,
						SectorID:   s.ID,
						MinerID:    t.minerID,
						ErrMsg:     ErrRetryExceed.Error(),
					}); err != nil {
						log.Error().Msgf("[Transformer] callback err: %s", err)
					}
					continue
				}
				var (
					target string
					srcURL string
				)

				target = fmt.Sprintf("%s/s-%s-%d", t.SealedDir, t.minerID, s.ID)
				srcURL = fmt.Sprintf("%ssealedsectors/%s/%d", t.downloadURL, t.minerID, s.ID)
				if _, err := os.Stat(target); err == nil {
					// file exist just retry download cache file
					log.Info().Msgf("[Transformer] target: %s exist, ignore", target)
				} else {
					log.Debug().Msgf("[Transformer] start download target: %s, src: %s", target, srcURL)
					d := InitDownloader(srcURL, target, "", t.token, t.minerID, t.transformPartSize, t.singleDownloadMaxWorkers, s.ID, false, true, t.ctx)
					if err := d.DownloadFile(); err != nil {
						log.Error().Msgf("[Transformer] Download sealed file failed, sector's metainfo: %+v, err: %s", s, err)
						// need retry
						go func() { t.ch <- s }()
						continue
					}
				}

				target = fmt.Sprintf("%s/s-%s-%d", t.workDir, t.minerID, s.ID)
				// we now support 32GB sector's download
				srcURL = fmt.Sprintf("%ssectortree/%s/32/%d", t.downloadURL, t.minerID, s.ID)

				if _, err := os.Stat(target); err == nil {
					// remove if exist
					log.Debug().Msgf("[Downloader] exist, rm: %s ", target)
					os.Remove(target)
				} else {
					log.Debug().Msgf("[Transformer] start download target: %s, src: %s", target, srcURL)
					d := InitDownloader(srcURL, target, t.CacheDir, t.token, t.minerID, t.transformPartSize, t.singleDownloadMaxWorkers, s.ID, true, false, t.ctx)
					if err := d.DownloadFile(); err != nil {
						log.Error().Msgf("[Transformer] DownloadFile cache failed, sector's metainfo: %+v, err: %s", s, err)
						// need retry
						go func() { t.ch <- s }()
						continue
					}
				}

				log.Info().Msgf("[Transformer] miner: %s, sector: %d download success, do callback", t.minerID, s.ID)
				if err := t.CallBack(DownloadCallBackContent{
					Action:     ActionDownload,
					Status:     StatusDownloadSuccessful,
					StatusCode: StatusCodeOK,
					SectorID:   s.ID,
					MinerID:    t.minerID,
				}); err != nil {
					log.Error().Msgf("[Transformer] callback err: %s", err)
				}

				// todo: add retry
				if err := t.DeclareSector(s.ID); err != nil {
					log.Error().Msgf("[Transformer] miner: %s DeclareSector: %d err: %s", t.minerID, s.ID, err)
					if err := t.CallBack(DownloadCallBackContent{
						Action:     ActionDeclare,
						Status:     StatusDeclareFailed,
						StatusCode: StatusCodeFailed,
						SectorID:   s.ID,
						MinerID:    t.minerID,
						ErrMsg:     err.Error(),
					}); err != nil {
						log.Error().Msgf("[Transformer] callback err: %s", err)
					}

					continue
				} else {
					if err := t.CallBack(DownloadCallBackContent{
						Action:     ActionDeclare,
						Status:     StatusDeclareSuccessful,
						StatusCode: StatusCodeOK,
						SectorID:   s.ID,
						MinerID:    t.minerID,
					}); err != nil {
						log.Error().Msgf("[Transformer] callback err: %s", err)
					}
				}
			case <-t.ctx.Done():
				return
			}
		}
	}()
}

func (t Transformer) DeclareSector(sectorID int) error {
	// file download successfully, need send declare request to lotus-miner
	if err := t.minerCli.SectorDeclare(sectorID, minerclient.FTSealed); err != nil {
		return err
	}

	if err := t.minerCli.SectorDeclare(sectorID, minerclient.FTCache); err != nil {
		return err
	}

	return nil
}

type DownloadCallBackContent struct {
	Action     string `json:"action,omitempty"`
	Status     string `json:"status,omitempty"`
	StatusCode int    `json:"statusCode,omitempty"`
	SectorID   int    `json:"sectorID,omitempty"`
	MinerID    string `json:"minerID,omitempty"`
	ErrMsg     string `json:"errMsg,omitempty"`
}

func (t Transformer) CallBack(content DownloadCallBackContent) error {
	c, err := json.Marshal(content)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", t.callBackURL, bytes.NewReader(c))
	if err != nil {
		return err
	}

	req.Header.Set("minerToken", t.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := t.cli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("Transformer CallBack err status: %d", resp.StatusCode)
	}

	return nil
}

type DownloadPart struct {
	start int64
	end   int64
}

// one download task <=> one Downloader
type Downloader struct {
	minerID  string
	sectorID int
	// 一次下载中最多开启 maxWorkers 个下载 goroutine
	maxWorkers int
	// worker 在下载时下载的分片大小
	partSize int
	cli      *http.Client
	// 文件在 GH 的 url
	srcFileURL string
	// 文件在本地的绝对路径
	targetFile string
	// targetPath 解压 cache 压缩文件需要
	targetPath    string
	downCh        chan DownloadPart
	decompression bool
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
	token         string
	depart        bool
}

// todo: too many params
func InitDownloader(downloadURL, targetFile, targetPath, token, minerID string, partSize, maxWorkers, sectorID int,
	decompression, depart bool, ctx context.Context) *Downloader {
	d := &Downloader{
		cli: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
			// todo: config this timeout
			// need increase if read response content timeout
			Timeout: time.Duration(10) * time.Minute,
		},
		maxWorkers:    5,
		partSize:      partSize,
		srcFileURL:    downloadURL,
		decompression: decompression,
		token:         token,
		wg:            sync.WaitGroup{},
		depart:        depart,
		targetFile:    targetFile,
		targetPath:    targetPath,
		minerID:       minerID,
		sectorID:      sectorID,
	}

	d.downCh = make(chan DownloadPart, 1024)
	d.ctx, d.cancel = context.WithCancel(ctx)

	return d
}

func (d *Downloader) startDownloadWorker() {
	defer func() {
		log.Debug().Msgf("[Downloader] worker finish task")
	}()

	for {
		select {
		case p, ok := <-d.downCh:
			log.Debug().Msgf("[Downloader] worker receive a download part: %+v", p)
			if !ok {
				log.Info().Msgf("[Downloader] worker's downCh is closed")
				return
			}

			if err := d.downloadRange(p.start, p.end); err != nil {
				log.Error().Msgf("[Downloader] retry download part: %+v, downloadRange err: %s\n", p, err)
				// retry until successfully
				go func() {
					d.downCh <- p
				}()
				continue
			}

			log.Debug().Msgf("[Downloader download part: %+v successfully", p)
			d.wg.Done()
		case <-d.ctx.Done():
			log.Debug().Msgf("[Downloader] worker's ctx done'")
			return
		}
	}
}

func (d *Downloader) downloadRange(start, end int64) error {
	// first get file's lenght and check the range.
	req, err := http.NewRequest("GET", d.srcFileURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Token", d.token) // need configurable
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
	resp, err := d.cli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return fmt.Errorf("range download errstatus: %d, body: %s", resp.StatusCode, string(b))
	}

	fd, err := os.OpenFile(d.targetFile, os.O_WRONLY, os.FileMode(0664))
	if err != nil {
		return err
	}
	defer fd.Close()

	_, err = fd.Seek(start, os.SEEK_SET)
	if err != nil {
		return err
	}

	_, err = io.Copy(fd, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

// need change when server support Head a file.
func (d *Downloader) headFile() (int64, error) {
	// first get file's lenght and check the range.
	req, err := http.NewRequest("GET", d.srcFileURL, nil)
	if err != nil {
		return -1, err
	}

	req.Header.Set("Token", d.token) // need configurable
	req.Header.Set("Range", "bytes=0-1")
	resp, err := d.cli.Do(req)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()

	s := strings.Split(resp.Header.Get("content-range"), "/")
	if len(s) < 2 {
		return -1, fmt.Errorf("range not supported")
	}

	len, _ := strconv.ParseInt(s[1], 10, 64)
	return len, nil
}

func (d *Downloader) scheduleDownload(len int64) {
	log.Debug().Msgf("[Downloader] start scheduleDownload")
	count := len / int64(d.partSize)
	start := int64(0)
	for i := int64(0); i < count; i++ {
		part := DownloadPart{
			start: int64(start),
			end:   start + int64(d.partSize) - 1,
		}

		d.wg.Add(1)
		d.downCh <- part
		start = part.end + 1
	}

	left := len % int64(d.partSize)
	if left > 0 {
		part := DownloadPart{
			start: start,
			end:   start + left - 1,
		}

		d.wg.Add(1)
		d.downCh <- part
	}

	log.Debug().Msgf("[Downloader] finish scheduleDownload")
}

func (d *Downloader) download() error {
	// first get file's lenght and check the range.
	req, err := http.NewRequest("GET", d.srcFileURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Token", d.token)
	resp, err := d.cli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fd, err := os.OpenFile(d.targetFile, os.O_WRONLY|os.O_CREATE, os.FileMode(0664))
	if err != nil {
		return err
	}
	_, err = io.Copy(fd, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func (d *Downloader) DownloadFile() error {
	if !d.depart {
		// todo: serverside should support multipart download for cache
		if _, err := os.Stat(d.targetFile); err == nil {
			// do nothing
			log.Debug().Msgf("[Downloader] targetfile: %s exist", d.targetFile)
		} else {
			if err := d.download(); err != nil {
				return err
			}
		}

		log.Info().Interface("src", d.srcFileURL).Interface("target", d.targetFile).Msgf("[Downloader] download successfully")
	} else {
		// create target file
		fd, err := os.OpenFile(d.targetFile, os.O_WRONLY|os.O_CREATE, os.FileMode(0664))
		if err != nil {
			return err
		}
		fd.Close()

		// multipart download
		len, err := d.headFile()
		if err != nil {
			return err
		}

		for i := 0; i < d.maxWorkers; i++ {
			log.Debug().Msgf("[Downloader] startDownloadWorker")
			go d.startDownloadWorker()
		}
		d.scheduleDownload(len)

		log.Info().Msgf("[Downloader] Waiting file downloaded")
		// wait download finish
		d.wg.Wait()
		log.Info().Interface("src", d.srcFileURL).Interface("target", d.targetFile).Msgf("[Downloader] download successfully")
	}

	// stop all workers, avoid gorounine leak
	d.cancel()

	// cache file need decompress
	if d.decompression {
		log.Info().Msgf("[Downloader] untar targetFile: %s, targetPath: %s", d.targetFile, d.targetPath)
		os.Mkdir(fmt.Sprintf("%s/s-%s-%d", d.targetPath, d.minerID, d.sectorID), os.FileMode(0664))
		if err := untar(d.targetFile, strings.TrimSuffix(d.targetPath, "cache")); err != nil {
			log.Error().Msgf("[Downloader] untar err: %s\n", err)
			// the file maybe broken, need retry
			return err
		}
	} else {
		// do nothing for sealed sector file here
	}

	return nil
}

func untar(tarball, target string) error {
	reader, err := os.Open(tarball)
	if err != nil {
		return err
	}
	defer reader.Close()
	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		path := filepath.Join(target, header.Name)
		info := header.FileInfo()
		if info.IsDir() {
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}
			continue
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(file, tarReader)
		if err != nil {
			return err
		}
	}
	return nil
}
