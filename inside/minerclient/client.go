package minerclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bitrainforest/PandaAgent/inside/config"
)

const (
	MethodFilecoinStorageDeclareSector = "Filecoin.StorageDeclareSector"
	MethodFilecoinStorageFindSector    = "Filecoin.StorageFindSector"
	DefaultID                          = 0
	// 32GB size sector
	SectorDefaultSize = 34359738368
)

// same as lotus source code. (https://github.com/filecoin-project/lotus/tree/master/storage/sealer/storiface/filetype.go#L11)
const (
	FTUnsealed SectorFileType = 1 << iota
	FTSealed
	FTCache
	FTUpdate
	FTUpdateCache

	FileTypes = iota
)

type SectorFileType int

func (t SectorFileType) String() string {
	switch t {
	case FTUnsealed:
		return "unsealed"
	case FTSealed:
		return "sealed"
	case FTCache:
		return "cache"
	case FTUpdate:
		return "update"
	case FTUpdateCache:
		return "update-cache"
	default:
		return fmt.Sprintf("<unknown %d>", t)
	}
}

type MinerCli struct {
	cli       *http.Client
	url       string
	apiToken  string
	id        int
	storageID string
}

// DeclareContent example
/*
'{
	"method": "Filecoin.StorageDeclareSector",
	"id": 0,
	"params": [
		"6b5bbb55-aaa2-4dec-8645-293b12c3d09c",
		{
			"Miner": 38310,
			"Number": 21
		},
		2,
		true
	]
}'
*/
type DeclareContent struct {
	Method    string        `json:"method"`
	DeclareID int           `json:"id"`
	Params    []interface{} `json:"params"`
}

type FindSectorContent struct {
	Method string        `json:"method"`
	ID     int           `json:"id"`
	Params []interface{} `json:"params"`
}

type FindSectorResult struct {
	JsonRPC string        `json:"jsonrpc"`
	Result  []interface{} `json:"result"`
}

type MetaInfo struct {
	Miner  int `json:"Miner"`
	Number int `json:"Number"`
}

func InitMinerCli(conf config.Config) MinerCli {
	id, _ := strconv.Atoi(strings.TrimPrefix(strings.TrimPrefix(conf.Miner.ID, "t"), "0"))
	return MinerCli{
		cli: &http.Client{
			Transport: &http.Transport{
				TLSHandshakeTimeout:   5 * time.Second,
				ResponseHeaderTimeout: 10 * time.Second,
				MaxIdleConns:          50,
				MaxIdleConnsPerHost:   1,
				MaxConnsPerHost:       10,
				IdleConnTimeout:       10 * time.Second,
			},
			Timeout: 10 * time.Second,
		},
		apiToken:  conf.Miner.APIToken,
		url:       conf.Miner.Address,
		id:        id,
		storageID: conf.Miner.StorageID,
	}
}

func (mc MinerCli) SectorFind(sectorID int, sft SectorFileType) (bool, error) {
	content := FindSectorContent{
		Method: MethodFilecoinStorageFindSector,
		ID:     DefaultID,
	}

	content.Params = append(content.Params, MetaInfo{
		Miner:  mc.id,
		Number: sectorID,
	})
	content.Params = append(content.Params, sft)
	content.Params = append(content.Params, SectorDefaultSize)
	content.Params = append(content.Params, true)

	res, err := json.Marshal(content)
	if err != nil {
		return false, err
	}

	req, err := http.NewRequest(http.MethodPost, mc.url, bytes.NewBuffer(res))
	if err != nil {
		return false, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", mc.apiToken)

	resp, err := mc.cli.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return false, fmt.Errorf("SectorFind err status: %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	findRes := FindSectorResult{}
	if err := json.Unmarshal(b, &findRes); err != nil {
		return false, err
	}

	return len(findRes.Result) > 0, nil
}

func (mc MinerCli) SectorDeclare(sectorID int, sft SectorFileType) error {
	content := DeclareContent{
		Method:    MethodFilecoinStorageDeclareSector,
		DeclareID: DefaultID,
	}

	content.Params = append(content.Params, mc.storageID)
	content.Params = append(content.Params, MetaInfo{
		Miner:  mc.id,
		Number: sectorID,
	})
	content.Params = append(content.Params, sft)
	content.Params = append(content.Params, true)

	res, err := json.Marshal(content)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, mc.url, bytes.NewBuffer(res))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", mc.apiToken)

	resp, err := mc.cli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("SectorDeclare err status: %d", resp.StatusCode)
	}

	exist, err := mc.SectorFind(sectorID, sft)
	if err != nil {
		return err
	}

	if !exist {
		return fmt.Errorf("SectorDeclare but not found")
	}

	return nil
}
