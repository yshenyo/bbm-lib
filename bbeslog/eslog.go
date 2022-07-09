package bbeslog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/google/uuid"
	"github.com/zjbobingtech/bbm-lib/utils"
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
				v.pushLogEs()
			}
		}
	}()
}

// PushLogs push log to es
// If indexName does not exist, use the default value
func PushLogs(logData string, addresses []string, options ...EsLogOption) {
	esLog := bbEsLog{
		Addresses: addresses,
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
