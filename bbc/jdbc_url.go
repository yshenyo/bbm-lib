package bbc

import (
	"fmt"
	"strings"
)

type DBConnection struct {
	ResourceID        int    `json:"resource_id"`
	DbType            string `json:"db_type"`
	IP                string `json:"ip"`
	DbPort            int    `json:"db_port"`
	DbName            string `json:"db_name"`
	DbUsername        string `json:"db_username"`
	DbPassword        string `json:"db_password"`
	DbVersion         string `json:"db_version"`
	DbInstanceName    string `json:"db_instance_name"`
	DbEncoding        string `json:"db_encoding"`
	JDBCType          string `json:"jdbc_type"`
	SshEnable         bool   `json:"ssh_enable"`
	ClusterName       string `json:"cluster_name"`
	NodeName          string `json:"node_name"`
	NodeType          string `json:"node_type"`
	SecurityMechanism string `json:"security_mechanism"`
}

func GetJDBCUrl(dc *DBConnection) string {
	return getJDBCUrl(dc)
}

func getJDBCUrl(dc *DBConnection) string {
	var url string
	switch strings.ToLower(dc.DbType) {
	case "oracle":
		url = fmt.Sprintf("jdbc:oracle:thin:@//%s:%d/%s", dc.IP, dc.DbPort, dc.DbName)
		if dc.JDBCType == "SID" {
			url = fmt.Sprintf("jdbc:oracle:thin:@%s:%d:%s", dc.IP, dc.DbPort, dc.DbName)
		}
		break
	case "db2":
		url = fmt.Sprintf("jdbc:db2://%s:%d/%s:blockingConnectionTimeout=30;", dc.IP, dc.DbPort, dc.DbName)
	case "mysql":
		url = fmt.Sprintf("jdbc:mysql://%s:%d/%s%s", dc.IP, dc.DbPort, dc.DbName, dc.DbEncoding)
	case "sqlserver":
		url = fmt.Sprintf("jdbc:sqlserver://%s:%d;", dc.IP, dc.DbPort)
		if dc.DbInstanceName != "" {
			url = fmt.Sprintf("jdbc:sqlserver://%s:%d;instanceName=%s;databaseName=%s", dc.IP, dc.DbPort, dc.DbInstanceName, dc.DbName)
		}
	case "postgres", "postgresql":
		url = fmt.Sprintf("jdbc:postgresql://%s:%d/%s", dc.IP, dc.DbPort, dc.DbName)
		break
	case "informix":
		url = fmt.Sprintf("jdbc:informix-sqli://%s:%d/%s:%s", dc.IP, dc.DbPort, dc.DbName, dc.DbEncoding)
		break
	case "dm":
		url = fmt.Sprintf("jdbc:dm://%s:%d/%s", dc.IP, dc.DbPort, dc.DbName)
		break
	case "gbase":
		url = fmt.Sprintf("jdbc:gbase://%s:%d/%s", dc.IP, dc.DbPort, dc.DbName)
		break
	case "shentong":
		url = fmt.Sprintf("jdbc:oscar://%s:%d/%s", dc.IP, dc.DbPort, dc.DbName)
		break
	case "oceanbase":
		url = fmt.Sprintf("jdbc:oceanbase://%s:%d/%s%s", dc.IP, dc.DbPort, dc.DbName, dc.DbEncoding)
		break
	case "hive":
		url = fmt.Sprintf("jdbc:hive://%s:%d/%s", dc.IP, dc.DbPort, dc.DbName)
	default:
		break
	}
	return url
}
