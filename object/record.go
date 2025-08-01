// Copyright 2023 The Casibase Authors. All Rights Reserved.
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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/beego/beego/context"
	"github.com/casibase/casibase/conf"
	"github.com/casibase/casibase/util"
)

var logPostOnly bool

func init() {
	logPostOnly = conf.GetConfigBool("logPostOnly")
}

type Record struct {
	Id int `xorm:"int notnull pk autoincr" json:"id"`

	Owner       string `xorm:"varchar(100) index" json:"owner"`
	Name        string `xorm:"varchar(100) index" json:"name"`
	CreatedTime string `xorm:"varchar(100)" json:"createdTime"`

	Organization string `xorm:"varchar(100)" json:"organization"`
	ClientIp     string `xorm:"varchar(100)" json:"clientIp"`
	UserAgent    string `xorm:"varchar(100)" json:"userAgent"`
	User         string `xorm:"varchar(100)" json:"user"`
	Method       string `xorm:"varchar(100)" json:"method"`
	RequestUri   string `xorm:"varchar(1000)" json:"requestUri"`
	Action       string `xorm:"varchar(1000)" json:"action"`
	Language     string `xorm:"varchar(100)" json:"language"`
	Region       string `xorm:"varchar(100)" json:"region"`
	City         string `xorm:"varchar(100)" json:"city"`
	Unit         string `xorm:"varchar(100)" json:"unit"`
	Section      string `xorm:"varchar(100)" json:"section"`

	Object   string `xorm:"mediumtext" json:"object"`
	Response string `xorm:"mediumtext" json:"response"`
	// ExtendedUser *User  `xorm:"-" json:"extendedUser"`

	Provider    string `xorm:"varchar(100)" json:"provider"`
	Block       string `xorm:"varchar(100)" json:"block"`
	BlockHash   string `xorm:"varchar(500)" json:"blockHash"`
	Transaction string `xorm:"varchar(500)" json:"transaction"`

	Provider2    string `xorm:"varchar(100)" json:"provider2"`
	Block2       string `xorm:"varchar(100)" json:"block2"`
	BlockHash2   string `xorm:"varchar(500)" json:"blockHash2"`
	Transaction2 string `xorm:"varchar(500)" json:"transaction2"`
	// For cross-chain records

	IsTriggered bool `json:"isTriggered"`
	NeedCommit  bool `json:"needCommit"`
}

type Response struct {
	Status string `json:"status"`
	Msg    string `json:"msg"`
}

func GetRecordCount(owner, field, value string) (int64, error) {
	session := GetSession(owner, -1, -1, field, value, "", "")
	return session.Count(&Record{Owner: owner})
}

func GetRecords(owner string) ([]*Record, error) {
	records := []*Record{}
	err := adapter.engine.Desc("id").Find(&records, &Record{Owner: owner})
	if err != nil {
		return records, err
	}

	return records, nil
}

func getAllRecords() ([]*Record, error) {
	records := []*Record{}
	err := adapter.engine.Desc("id").Find(&records, &Record{})
	if err != nil {
		return records, err
	}

	return records, nil
}

func GetPaginationRecords(owner string, offset, limit int, field, value, sortField, sortOrder string) ([]*Record, error) {
	records := []*Record{}
	session := GetSession(owner, offset, limit, field, value, sortField, sortOrder)
	err := session.Find(&records)
	if err != nil {
		return records, err
	}

	return records, nil
}

func getRecord(owner string, name string) (*Record, error) {
	if owner == "" || name == "" {
		return nil, nil
	}

	record := Record{Name: name}
	existed, err := adapter.engine.Get(&record)
	if err != nil {
		return &record, err
	}

	if existed {
		return &record, nil
	} else {
		return nil, nil
	}
}

func GetRecord(id string) (*Record, error) {
	owner, name := util.GetOwnerAndNameFromIdNoCheck(id)
	return getRecord(owner, name)
}

func UpdateRecord(id string, record *Record) (bool, error) {
	owner, name := util.GetOwnerAndNameFromId(id)
	p, err := getRecord(owner, name)
	if err != nil {
		return false, err
	} else if p == nil {
		return false, nil
	}

	// Update provider
	if record.Provider != p.Provider {
		record.Block = ""
		record.BlockHash = ""
		record.Transaction = ""
	}
	if record.Provider2 != p.Provider2 {
		record.Block2 = ""
		record.BlockHash2 = ""
		record.Transaction2 = ""
	}

	affected, err := adapter.engine.Where("name = ?", name).AllCols().Update(record)
	if err != nil {
		return false, err
	}

	return affected != 0, nil
}

func UpdateRecordInternal(id int, record Record) error {
	_, err := adapter.engine.ID(id).Update(record)
	if err != nil {
		return err
	}
	return nil
}

func UpdateRecordFields(id string, fields map[string]interface{}) (bool, error) {
	owner, name := util.GetOwnerAndNameFromId(id)
	if p, err := getRecord(owner, name); err != nil {
		return false, err
	} else if p == nil {
		return false, nil
	}

	affected, err := adapter.engine.Table(&Record{}).Where("name = ?", name).Update(fields)
	if err != nil {
		return false, err
	}

	return affected != 0, nil
}

func NewRecord(ctx *context.Context) (*Record, error) {
	ip := strings.Replace(util.GetIPFromRequest(ctx.Request), ": ", "", -1)
	action := strings.Replace(ctx.Request.URL.Path, "/api/", "", -1)
	requestUri := util.FilterQuery(ctx.Request.RequestURI, []string{"accessToken"})
	if len(requestUri) > 1000 {
		requestUri = requestUri[0:1000]
	}

	object := ""
	if ctx.Input.RequestBody != nil && len(ctx.Input.RequestBody) != 0 {
		object = string(ctx.Input.RequestBody)
	}

	respBytes, err := json.Marshal(ctx.Input.Data()["json"])
	if err != nil {
		return nil, err
	}

	var resp Response
	err = json.Unmarshal(respBytes, &resp)
	if err != nil {
		return nil, err
	}

	language := ctx.Request.Header.Get("Accept-Language")
	if len(language) > 2 {
		language = language[0:2]
	}
	languageCode := conf.GetLanguage(language)

	// get location info from client ip
	locationInfo, err := util.GetInfoFromIP(ip)
	if err != nil {
		return nil, err
	}
	region := locationInfo.Country
	city := locationInfo.City
	if err != nil {
		return nil, err
	}

	record := Record{
		Name:        util.GenerateId(),
		CreatedTime: util.GetCurrentTime(),
		ClientIp:    ip,
		User:        "",
		Method:      ctx.Request.Method,
		RequestUri:  requestUri,
		Action:      action,
		Language:    languageCode,
		Region:      region,
		City:        city,
		Object:      object,
		Response:    fmt.Sprintf("{\"status\":\"%s\",\"msg\":\"%s\"}", resp.Status, resp.Msg),
		IsTriggered: false,
	}
	return &record, nil
}

func AddRecord(record *Record) (bool, interface{}, error) {
	if logPostOnly && record.Method == "GET" {
		return false, nil, nil
	}

	if strings.HasSuffix(record.Action, "-record") {
		return false, nil, nil
	}

	if strings.HasSuffix(record.Action, "-record-second") {
		return false, nil, nil
	}

	if record.Provider == "" {
		providerFrist, providerSecend, err := GetTwoActiveBlockchainProvider("admin")
		if err != nil {
			return false, nil, err
		}

		if providerFrist != nil {
			record.Provider = providerFrist.Name
		}

		if providerSecend != nil {
			record.Provider2 = providerSecend.Name
		}
	}

	record.Owner = record.Organization

	affected, err := adapter.engine.Insert(record)
	if err != nil {
		return false, nil, err
	}

	if record.NeedCommit {
		affected2, data, err := CommitRecord(record)
		if err != nil {
			return false, nil, err
		}

		return affected2, data, nil
	}

	return affected != 0, nil, nil
}

func DeleteRecord(record *Record) (bool, error) {
	affected, err := adapter.engine.Where("name = ?", record.Name).Delete(&Record{})
	if err != nil {
		return false, err
	}

	return affected != 0, nil
}

func (record *Record) getId() string {
	return fmt.Sprintf("%s/%s", record.Owner, record.Name)
}
