package downloader

import (
	"archive/tar"
	"bytes"
	"context"
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

	"github.com/pandarua-agent/inside/checker"
	"github.com/pandarua-agent/inside/config"
	"github.com/pandarua-agent/inside/minerclient"
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
	}
	t.ch = make(chan checker.Sector, t.MaxDownloader)
	t.ctx, t.cancel = context.WithCancel(ctx)

	log.Info().Msgf("[Transformer] init: %+v", t)

	return t
}

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

				target := fmt.Sprintf("%s/s-%s-%d", t.SealedDir, t.minerID, s.ID)
				srcURL := fmt.Sprintf("%s/sealedsectors/%s/%d", t.downloadURL, t.minerID, s.ID)
				if _, err := os.Stat(target); err == nil {
					// file exist just retry download cache file
					log.Info().Msgf("[Transformer] target: %s exist, ignore", target)
				} else {
					log.Debug().Msgf("[Transformer] start download target: %s, src: %s", target, srcURL)
					d := InitDownloader(srcURL, target, t.transformPartSize, t.singleDownloadMaxWorkers, false, t.ctx)
					if err := d.DownloadFile(); err != nil {
						log.Error().Msgf("[Transformer] DownloadFile sector's metainfo: %+v, err: %s", s, err)
						// need retry
						go func() { t.ch <- s }()
						continue
					}
				}

				target = fmt.Sprintf("%s/s-%s-%d", t.CacheDir, t.minerID, s.ID)
				// we now support 32GB sector's download
				srcURL = fmt.Sprintf("%s/sectortree/%s/32/%d", t.downloadURL, t.minerID, s.ID)
				log.Debug().Msgf("[Transformer] start download target: %s, src: %s", target, srcURL)
				d := InitDownloader(srcURL, target, t.transformPartSize, t.singleDownloadMaxWorkers, true, t.ctx)
				if err := d.DownloadFile(); err != nil {
					log.Error().Msgf("[Transformer] DownloadFile sector's metainfo: %+v, err: %s", s, err)
					// need retry
					go func() { t.ch <- s }()
					continue
				}

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
				}

				log.Info().Msgf("[Transformer] miner: %s, sector: %d download and declare success, do callback", t.minerID, s.ID)
				if err := t.CallBack(DownloadCallBackContent{
					Action:     ActionDownload,
					Status:     StatusDownloadSuccessful,
					StatusCode: StatusCodeOK,
					SectorID:   s.ID,
					MinerID:    t.minerID,
				}); err != nil {
					log.Error().Msgf("[Transformer] callback err: %s", err)
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
	Action     string `json:action",omitempty"`
	Status     string `json:status",omitempty"`
	StatusCode int    `json:int",omitempty"`
	SectorID   int    `json:sectorID",omitempty"`
	MinerID    string `json:inerID",omitempty"`
	ErrMsg     string `json:errMsg",omitempty"`
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
	// 一次下载中最多开启 maxWorkers 个下载 goroutine
	maxWorkers int
	// worker 在下载时下载的分片大小
	partSize int
	cli      *http.Client
	// 文件在 GH 的 url
	srcFileURL string
	// 文件在本地的绝对路径
	targetFile string
	// 解压 cache 压缩文件需要
	targetPath    string
	downCh        chan DownloadPart
	decompression bool
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
}

func InitDownloader(downloadURL, targetFile string, partSize, maxWorkers int, decompression bool, ctx context.Context) Downloader {
	d := Downloader{
		cli: &http.Client{
			Timeout: time.Duration(5) * time.Second,
		},
		maxWorkers:    5,
		partSize:      partSize,
		srcFileURL:    downloadURL,
		targetFile:    targetFile,
		decompression: decompression,
	}

	d.downCh = make(chan DownloadPart, d.maxWorkers)
	d.ctx, d.cancel = context.WithCancel(ctx)

	return d
}

func (d Downloader) startDownloadWorker() {
	for {
		select {
		case p, ok := <-d.downCh:
			if !ok {
				break
			}

			if err := d.downloadRange(p.start, p.end); err != nil {
				fmt.Printf("[ERR] Downloader downloadRange err: %s\n", err)
				// todo: limit the retry
				go func() { d.downCh <- p }()
			} else {
				d.wg.Done()
			}
		case <-d.ctx.Done():
			break
		}
	}
}

func (d Downloader) downloadRange(start, end int64) error {
	// first get file's lenght and check the range.
	req, err := http.NewRequest("GET", d.srcFileURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Token", "abc") // need configurable
	req.Header.Set("Range", "bytes=0-1")
	resp, err := d.cli.Do(req)
	if err != nil {
		return err
	}

	fd, err := os.OpenFile(d.targetFile, os.O_WRONLY, os.FileMode(0664))
	// todo: fd seek here may have problem, test it
	_, err = fd.Seek(start, os.SEEK_SET)
	if err != nil {
		return err
	}

	io.Copy(fd, resp.Body)

	return nil
}

// need change when server support Head a file.
func (d Downloader) headFile() (int64, error) {
	// first get file's lenght and check the range.
	req, err := http.NewRequest("GET", d.srcFileURL, nil)
	if err != nil {
		return -1, err
	}

	req.Header.Set("Token", "abc") // need configurable
	req.Header.Set("Range", "bytes=0-1")
	resp, err := d.cli.Do(req)
	if err != nil {
		return -1, err
	}

	lenStr := strings.Split(resp.Header.Get("content-range"), "/")[1]
	len, _ := strconv.ParseInt(lenStr, 10, 64)
	return len, nil
}

func (d Downloader) scheduleDownload(len int64) {
	fmt.Printf("[debug] Downloader start scheduleDownload\n")
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

	fmt.Printf("[debug] Downloader finish scheduleDownload\n")
}

func (d Downloader) DownloadFile() error {
	len, err := d.headFile()
	if err != nil {
		return err
	}

	// create target file
	fd, err := os.OpenFile(d.targetFile, os.O_WRONLY|os.O_CREATE, os.FileMode(0664))
	if err != nil {
		return err
	}
	fd.Close()

	for i := 0; i < d.maxWorkers; i++ {
		go d.startDownloadWorker()
	}

	d.scheduleDownload(len)
	// wait download finish
	d.wg.Wait()
	// stop all workers
	d.cancel()

	// cache file need decompress
	if d.decompression {
		if err := untar(d.targetFile, d.targetPath); err != nil {
			fmt.Printf("[debug] downloader untar err: %s\n", err)
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
