package graph

import (
	"bytes"
	"encoding/json"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"io/ioutil"
	"net/http"
	"time"
)

type History struct {
	Outputs []Tx
	Spends  []Tx
}

func GetHistory(url string, address *wallet.Addr, startHeight int64) (*History, error) {
	jsonData := map[string]interface{}{
		"query": HistoryQuery,
		"variables": map[string]interface{}{
			"address": address.String(),
			"height":  startHeight,
		},
	}
	jsonValue, err := json.Marshal(jsonData)
	if err != nil {
		return nil, jerr.Get("error marshaling json for get history", err)
	}
	request, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonValue))
	if err != nil {
		return nil, jerr.Get("error creating new request for get history", err)
	}
	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: time.Second * 10}
	response, err := client.Do(request)
	if err != nil {
		return nil, jerr.Get("error the HTTP request failed", err)
	}
	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, jerr.Get("error reading response body", err)
	}
	var dataStruct = struct {
		Data struct {
			Address Address `json:"address"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}{}
	if err := json.Unmarshal(data, &dataStruct); err != nil {
		return nil, jerr.Get("error unmarshalling json", err)
	}
	if len(dataStruct.Errors) > 0 {
		return nil, jerr.Get("error response data", jerr.New(dataStruct.Errors[0].Message))
	}
	var history = new(History)
OutputLoop:
	for _, output := range dataStruct.Data.Address.Outputs {
		for _, tx := range history.Outputs {
			if tx.Hash == output.Tx.Hash {
				continue OutputLoop
			}
		}
		history.Outputs = append(history.Outputs, output.Tx)
	}
SpendsLoop:
	for _, input := range dataStruct.Data.Address.Spends {
		for _, tx := range history.Spends {
			if tx.Hash == input.Tx.Hash {
				continue SpendsLoop
			}
		}
		history.Spends = append(history.Spends, input.Tx)
	}
	return history, nil
}

const QueryTx = `tx {
	hash
	seen
	raw
	inputs {
		index
		prev_hash
		prev_index
	}
	outputs {
		index
		amount
		lock {
			address
		}
		spends {
			tx {
				hash
				seen
				raw
				inputs {
					index
					prev_hash
					prev_index
				}
				outputs {
					index
					amount
					lock {
						address
					}
				}
				blocks {
					hash
					timestamp
					height
				}
			}
		}
	}
	blocks {
		hash
		timestamp
		height
	}
}`

const HistoryQuery = `query ($address: String!, $height: Int) {
	address (address: $address) {
		outputs(height: $height) {
			` + QueryTx + `
		}
		spends(height: $height) {
			` + QueryTx + `
		}
	}
}`
