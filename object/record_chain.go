// Copyright 2024 The Casibase Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package object

import (
	"fmt"

	"github.com/casibase/casibase/chain"
	"github.com/casibase/casibase/util"
)

type Param struct {
	Key   string `json:"key"`
	Field string `json:"field"`
	Value string `json:"value"`
}

func (record *Record) getRecordProvider(chainProvider string) (*Provider, error) {
	if chainProvider != "" {
		provider, err := getProvider("admin", chainProvider)
		if err != nil {
			return nil, err
		}

		if provider == nil {
			return nil, fmt.Errorf("the blockchain provider: %s is not found", chainProvider)
		}

		return provider, nil
	}

	provider, err := GetActiveBlockchainProvider("admin")
	if err != nil {
		return nil, err
	}

	return provider, nil
}

func (record *Record) getRecordChainClient(chainProvider string) (chain.ChainClientInterface, *Provider, error) {
	provider, err := record.getRecordProvider(chainProvider)
	if err != nil {
		return nil, nil, err
	}
	if provider == nil {
		return nil, nil, fmt.Errorf("there is no active blockchain provider")
	}

	client, err := chain.NewChainClient(provider.Type, provider.ClientId, provider.ClientSecret, provider.Region, provider.Network, provider.Chain, provider.ProviderUrl, provider.Text, provider.UserKey, provider.UserCert, provider.SignKey, provider.SignCert, provider.ContractName, provider.ContractMethod)
	if err != nil {
		return nil, nil, err
	}

	return client, provider, nil
}

func (record *Record) toMap() map[string]string {
	result := map[string]string{}

	result["id"] = fmt.Sprintf("%d", record.Id)
	result["owner"] = record.Owner
	result["name"] = record.Name
	result["createdTime"] = record.CreatedTime

	result["organization"] = record.Organization
	result["clientIp"] = record.ClientIp
	result["user"] = record.User
	result["method"] = record.Method
	result["requestUri"] = record.RequestUri
	result["action"] = record.Action
	result["language"] = record.Language

	result["object"] = record.Object
	result["response"] = record.Response

	result["provider"] = record.Provider
	result["block"] = record.Block
	result["isTriggered"] = fmt.Sprintf("%t", record.IsTriggered)

	return result
}

func (record *Record) toParam() string {
	record2 := *record
	record2.Provider = ""
	record2.Block = ""
	record2.Transaction = ""
	record2.BlockHash = ""
	record2.Provider2 = ""
	record2.Block2 = ""
	record2.Transaction2 = ""
	record2.BlockHash2 = ""

	res := Param{
		Key:   record2.getId(),
		Field: "Record",
		Value: util.StructToJson(record2),
	}
	return util.StructToJson(res)
}

func CommitRecord(record *Record) (bool, interface{}, error) {
	if record.Block != "" {
		return false, nil, fmt.Errorf("the record: %s has already been committed, blockId = %s", record.getId(), record.Block)
	}

	client, provider, err := record.getRecordChainClient(record.Provider)
	if err != nil {
		return false, nil, err
	}
	record.Provider = provider.Name

	blockId, transactionId, blockHash, err := client.Commit(record.toParam())
	if err != nil {
		return false, nil, err
	}

	data := map[string]interface{}{
		"provider":    record.Provider,
		"block":       blockId,
		"transaction": transactionId,
		"block_hash":  blockHash,
	}

	// Update the record fields to avoid concurrent update race conditions
	affected, err := UpdateRecordFields(record.getId(), data)
	return affected, data, err
}

func CommitRecordSecond(record *Record) (bool, error) {
	if record.Block2 != "" {
		return false, fmt.Errorf("the record: %s has already been committed, blockId = %s", record.getId(), record.Block2)
	}

	client, provider, err := record.getRecordChainClient(record.Provider2)
	if err != nil {
		return false, err
	}
	record.Provider2 = provider.Name

	blockId, transactionId, blockHash, err := client.Commit(record.toParam())
	if err != nil {
		return false, err
	}

	// Update the record fields to avoid concurrent update race conditions
	return UpdateRecordFields(record.getId(), map[string]interface{}{
		"provider2":    record.Provider2,
		"block2":       blockId,
		"transaction2": transactionId,
		"block_hash2":  blockHash,
	})
}

func QueryRecord(id string) (string, error) {
	record, err := GetRecord(id)
	if err != nil {
		return "", err
	}
	if record == nil {
		return "", fmt.Errorf("the record: %s does not exist", id)
	}

	if record.Block == "" {
		return "", fmt.Errorf("the record: %s's block ID should not be empty", record.getId())
	}

	client, _, err := record.getRecordChainClient(record.Provider)
	if err != nil {
		return "", err
	}

	res, err := client.Query(record.Transaction, record.toParam())
	if err != nil {
		return "", err
	}

	return res, nil
}

func QueryRecordSecond(id string) (string, error) {
	record, err := GetRecord(id)
	if err != nil {
		return "", err
	}
	if record == nil {
		return "", fmt.Errorf("the record: %s does not exist", id)
	}

	if record.Block2 == "" {
		return "", fmt.Errorf("the record: %s's block ID should not be empty", record.getId())
	}

	client, _, err := record.getRecordChainClient(record.Provider2)
	if err != nil {
		return "", err
	}

	res, err := client.Query(record.Transaction2, record.toParam())
	if err != nil {
		return "", err
	}

	return res, nil
}
