package promethus

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"go.uber.org/zap"
)

type Prometheus struct {
}

type PrometheusResult struct {
	Status    string         `json:"status"`
	Data      PrometheusData `json:"data"`
	ErrorType string         `json:"errorType"`
	Error     string         `json:"error"`
}
type PrometheusData struct {
	ResultType string         `json:"resultType"`
	Result     []MetricResult `json:"result"`
}

type MetricResult struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value"`
	Values [][]interface{}   `json:"values"`
}

func (prom PrometheusResult) MarshalBinary() ([]byte, error) {
	return json.Marshal(prom)
}

func (prom PrometheusResult) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &prom)
}

var globalTransport *http.Transport

func init() {
	globalTransport = &http.Transport{}
}

func (prom Prometheus) Query(class string, query string, monitorServerUrl string) (result PrometheusResult, err error) {
	urlStr := fmt.Sprintf("%s/api/v1/%s?%s", monitorServerUrl, class, query)
	client := http.Client{
		Transport: globalTransport,
	}
	res, err := client.Get(urlStr)
	if err != nil {
		return result, fmt.Errorf("query from monitor error %v %v", zap.String("url", urlStr), zap.String("err", err.Error()))
	}
	rbody, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return result, fmt.Errorf("query from monitor error %v %v", zap.String("url", urlStr), zap.String("err", err.Error()))
	}
	err = json.Unmarshal(rbody, &result)
	if err != nil {
		return result, fmt.Errorf("query from monitor error %v %v", zap.String("url", urlStr), zap.String("err", err.Error()))
	}
	return result, nil
}
