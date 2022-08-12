package minerclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/pandarua-agent/inside/config"
)

const (
	MethodFilecoinStorageDeclareSector = "Filecoin.StorageDeclareSector"
	DefaultDeclareID                   = 0
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
	target    string
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
	Method    string        `json:method",omitempty"`
	DeclareID int           `json:id",omitempty"`
	Params    []interface{} `json:params",omitempty"`
}

type MetaInfo struct {
	Miner  int `json:Miner",omitempty"`
	Number int `json:Number",omitempty"`
}

func InitMinerCli(conf config.Config) MinerCli {
	id, _ := strconv.Atoi(strings.TrimPrefix(strings.TrimPrefix(conf.Miner.ID, "t"), "0"))
	return MinerCli{
		cli:       http.DefaultClient,
		apiToken:  conf.Miner.APIToken,
		id:        id,
		storageID: conf.Miner.StorageID,
	}
}

// todo: bugfix
func (mc MinerCli) SectorDeclare(sectorID int, sft SectorFileType) error {
	content := DeclareContent{
		Method:    MethodFilecoinStorageDeclareSector,
		DeclareID: DefaultDeclareID,
	}

	content.Params = append(content.Params, mc.storageID)
	// Note(freddie): need fix
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

	req, err := http.NewRequest(http.MethodPost, mc.target, bytes.NewBuffer(res))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", mc.apiToken)

	resp, err := mc.cli.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("SectorDeclare err status: %d", resp.StatusCode)
	}

	return nil
}
