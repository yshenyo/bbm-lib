package bbetcd

import "fmt"

type ServerData struct {
	ServerName string
	Host       string
	Port       int
}

const BBMEtcdSchema = "bbm_etcd"

func GetEtcdKey(serverName, serverHost string, serverPort int) string {
	return fmt.Sprintf("%v/%v/%v/%v", BBMEtcdSchema, serverName, serverHost, serverPort)
}

func GetEtcdPrefix(serverName string) string {
	return fmt.Sprintf("%v/%v", BBMEtcdSchema, serverName)
}
