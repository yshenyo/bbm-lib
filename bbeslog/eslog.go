package bbeslog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/google/uuid"
	"github.com/zjbobingtech/bbm-lib/utils"
	"github.com/zjbobingtech/bbm-lib/utils/connect"
)

// elasticsearch:7.4.2
// kibana:7.4.2

type EsLogOption struct {
	UserName  string
	Password  string
	IndexName string
}

const esLogIndexPre = "bbm_log_"

var logChan = make(chan bbEsLog, 1023)

func init() {
	go func() {
		for {
			select {
			case v := <-logChan:
				go v.pushLogEs()
			}
		}
	}()
}

// PushLogs push log to es
// If indexName does not exist, use the default value
// Addresses ps: http://127.0.0.1:9200
func PushLogs(logData string, addresses []string, options ...EsLogOption) {
	var usedAddresses []string
	for _, addr := range addresses {
		var telAddres string
		if strings.Contains(addr, "http://") {
			telAddres = strings.Replace(addr, "http://", "", -1)
		} else if strings.Contains(addr, "https://") {
			telAddres = strings.Replace(addr, "https://", "", -1)
		}
		if telAddres == "" {
			continue
		}
		if err := connect.TelnetAddress(telAddres); err == nil {
			usedAddresses = append(usedAddresses, addr)
		}
	}
	if len(usedAddresses) == 0 {
		return
	}
	esLog := bbEsLog{
		Addresses: usedAddresses,
	}
	esLog.IndexName = esLogIndexPre + utils.GetLocalIp()
	if len(options) != 0 {
		esLog.UserName = options[0].UserName
		esLog.Password = options[0].Password
		if options[0].IndexName != "" {
			esLog.IndexName = options[0].IndexName
		}
	}
	esLog.LogData = logData
	logChan <- esLog
	return
}

type bbEsLog struct {
	UserName  string
	Password  string
	IndexName string
	Addresses []string
	LogData   string
}

func (es bbEsLog) pushLogEs() {
	logData := es.LogData
	cli, err := elasticsearch.NewClient(
		elasticsearch.Config{
			Password:  es.Password,
			Username:  es.UserName,
			Addresses: es.Addresses,
		},
	)
	if err != nil {
		fmt.Printf("new es client error %v \n", err)
		return
	}

	//logDta if is json
	var data []byte
	if json.Valid([]byte(logData)) {
		var dat map[string]interface{}
		_ = json.Unmarshal([]byte(logData), &dat)
		dat["source_ip"] = utils.GetLocalIp()
		data, _ = json.Marshal(dat)
	} else {
		type LogData struct {
			IpSource string      `json:"source_ip"`
			Time     string      `json:"time"`
			LogData  interface{} `json:"log_data"`
		}
		dat := LogData{}
		dat.IpSource = utils.GetLocalIp()
		dat.Time = time.Now().Format("2006-01-02 15:04:05")
		dat.LogData = logData
		data, _ = json.Marshal(dat)
	}
	req := esapi.IndexRequest{
		Index:      es.IndexName,
		DocumentID: uuid.New().String(),
		Body:       bytes.NewReader(data[:]),
		Refresh:    "true",
	}
	res, err := req.Do(context.Background(), cli)
	defer res.Body.Close()
	if err != nil {
		fmt.Printf("request es error %v \n", err)
		return
	}
	return
}
