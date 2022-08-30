package boost

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	MethodFilecoinBoostDeal = "Filecoin.BoostDeal"
)

type Deal struct {
	UUID                string `json:"uuid,omitempty"`
	PieceID             string `json:"pieceCID,omitempty"`
	ClientPeerID        string `json:"clientPeerID,omitempty"`
	PieceSize           int64  `json:"pieceSize,omitempty"`
	Verified            bool   `json:"verified,omitempty"`
	SignedProposalCID   string `json:"signedProposalCID,omitempty"`
	DealDataRootCID     string `json:"dealDataRootCID,omitempty"`
	ClientAddress       string `json:"clientAddress,omitempty"`
	CreatedAt           string `json:"createdAt,omitempty"`
	ClientSignature     int    `json:"clientSignature,omitempty"`
	ClientSignatureData string `json:"clientSignatureData,omitempty"`
}

type BoostCli struct {
	cli        *http.Client
	url        string
	graphQlURL string
	apiToken   string
	ch         chan *Deal
}

func InitBoostCli(url, graphQlURL, token string, ch chan *Deal) *BoostCli {
	return &BoostCli{
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
		url:        url,
		graphQlURL: graphQlURL,
		apiToken:   token,
		ch:         ch,
	}
}

// todo: use json-rpc tu marshal
type BoostDealReq struct {
	Method string        `json:"method,omitempty"`
	Params []interface{} `json:"params,omitempty"`
	ID     string        `json:"id,omitempty"`
}

type BoostDealResp struct {
	Result struct {
		DealUuid           string `json:"DealUuid,omitempty"`
		CreatedAt          string `json:"CreatedAt,omitempty"`
		ClientDealProposal struct {
			Proposal struct {
				PieceCID struct {
					Root string `json:"/,omitempty"`
				} `json:"PieceCID,omitempty"`
				PieceSize    int    `json:"PieceSize,omitempty"`
				VerifiedDeal bool   `json:"VerifiedDeal,omitempty"`
				Client       string `json:"Client,omitempty"`
				Provider     string `json:"Provider,omitempty"`
			} `json:"Proposal,omitempty"`
			ClientSignature struct {
				Type int    `json:"type,omitempty"`
				Data string `json:"Data,omitempty"`
			} `json:"ClientSignature,omitempty"`
		} `json:"ClientDealProposal,omitempty"`
		ClientPeerID string `json:"ClientPeerID,omitempty"`
		DealDataRoot struct {
			Root string `json:"/,omitempty"`
		} `json:"DealDataRoot,omitempty"`
	} `json:"result,omitempty"`
}

// send json-rpc Filecoin.BoostDeal to boostd server
// todo:
func (bc *BoostCli) GetBoostDeal(id string) (*Deal, error) {
	bdq := BoostDealReq{
		Method: MethodFilecoinBoostDeal,
		ID:     "0",
	}

	bdq.Params = append(bdq.Params, id)
	res, err := json.Marshal(bdq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, bc.url, bytes.NewBuffer(res))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bc.apiToken)

	resp, err := bc.cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("%s err status: %d", MethodFilecoinBoostDeal, resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	dealRes := BoostDealResp{}
	if err := json.Unmarshal(b, &dealRes); err != nil {
		return nil, err
	}

	deal := Deal{
		UUID:              dealRes.Result.DealUuid,
		PieceID:           dealRes.Result.ClientDealProposal.Proposal.PieceCID.Root,
		ClientPeerID:      dealRes.Result.ClientPeerID,
		PieceSize:         int64(dealRes.Result.ClientDealProposal.Proposal.PieceSize),
		Verified:          dealRes.Result.ClientDealProposal.Proposal.VerifiedDeal,
		SignedProposalCID: dealRes.Result.ClientDealProposal.Proposal.PieceCID.Root,
		DealDataRootCID:   dealRes.Result.DealDataRoot.Root,
		ClientAddress:     dealRes.Result.ClientPeerID,
		CreatedAt:         dealRes.Result.CreatedAt,
		// todo: remember need fix or change
		ClientSignature:     dealRes.Result.ClientDealProposal.ClientSignature.Type,
		ClientSignatureData: dealRes.Result.ClientDealProposal.ClientSignature.Data,
	}

	return &deal, nil
}

type DealID struct {
	ID string
}

type DetailDeal struct {
	ID string `json:"ID,omitempty"`
}

type GraphQlRes struct {
	Data struct {
		Deals struct {
			Deals []DetailDeal `json:"deals,omitempty"`
		} `json:"deals,omitempty"`
	} `json:"data,omitempty"`
}

// todo:
func (bc *BoostCli) GraphQl(offset, limit int) ([]DealID, error) {
	query := fmt.Sprintf(`{"query":"query { deals(offset: %d, limit: %d) { deals { ID CreatedAt PieceCid } } }"}`, offset, limit)
	req, err := http.NewRequest(http.MethodPost, bc.url, bytes.NewBufferString(query))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := bc.cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("BoostCli GraphQl err status: %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	graphRes := GraphQlRes{}
	if err := json.Unmarshal(b, &graphRes); err != nil {
		return nil, err
	}

	res := make([]DealID, 0, limit)
	for _, v := range graphRes.Data.Deals.Deals {
		res = append(res, DealID{
			ID: v.ID,
		})
	}

	return res, nil
}

// query loop
func (bc *BoostCli) Start() {
	defer log.Warn().Msgf("[BoostCli] exit")

	off := 0
	limit := 50
	for {
		res, err := bc.GraphQl(off, limit)
		if err != nil {
			log.Error().Msgf("[BoostCli] GraphQl off: %d, limit: %d, err: %s", off, limit, err)
			continue
		}

		off += len(res)
		for i := 0; i < len(res); i++ {
			deal, err := bc.GetBoostDeal(res[i].ID)
			if err != nil {
				log.Error().Msgf("[BoostCli] GetBoostDeal %d, err: %s", res[i].ID, err)
				continue
			}

			bc.ch <- deal
		}

		// maybe need configurable
		time.Sleep(5 * time.Second)
	}
}
