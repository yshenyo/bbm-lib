package bbc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"github.com/spf13/cast"
	"github.com/zjbobingtech/bbm_lib/utils/encryption"
	"go.uber.org/zap"
	"gorm.io/datatypes"
)

type CommonBBC struct {
	Request  QueryRequest
	Response QueryResultOK
	Error    QueryResultErr
	Instance Instance
	Class    string
	Url      string
	Analyze  []analyzeResult
	Timeout  int
}

type CommonBbcRequest struct {
	RequestID string `json:"request_id"`
}

func NewCommonBBC(url string) *CommonBBC {
	return &CommonBBC{
		Url: url,
	}
}

func (bbc *CommonBBC) GetQueryData() {
	con := bbc.Instance.Account[1].Content
	bbc.Request.Database.Type = bbc.Instance.Category[1]

	switch bbc.Request.Database.Type {
	case "oceanbase":
		if con.NodeName != "" && con.ClusterName != "" {
			bbc.Request.Database.Username = fmt.Sprintf("%s@%s#%s", con.DbUsername, con.NodeName, con.ClusterName)
		} else {
			bbc.Request.Database.Username = con.DbUsername
		}
	case "tidb":
		bbc.Request.Database.Type = "mysql"
		bbc.Request.Database.Username = con.DbUsername
	default:
		bbc.Request.Database.Username = con.DbUsername
	}
	//bbc.Request.Database.Username = con.DbUsername
	//var er error
	//bbc.Request.Database.Password, er = encryption.Decode_Nologging(con.DbPassword)
	//if er != nil {
	//	bbc.Request.Database.Password = con.DbPassword
	//}
	p, err := encryption.Decode(con.DbPassword)
	if err == nil && p != "" {
		con.DbPassword = p
	}
	bbc.Request.Database.Password = con.DbPassword
	bbc.Request.Database.Version = con.DbVersion
	if bbc.Request.Database.Type == "db2" {
		if con.SecurityMechanism != "" {
			bbc.Request.Database.SecurityMechanism = con.SecurityMechanism
		}
	}
	bbc.getQueryURL()
	return
}

func (bbc *CommonBBC) getQueryDataWithAccount(username, password string) (data QueryRequest) {
	con := bbc.Instance.Account[1].Content
	bbc.Request.Database.Type = bbc.Instance.Category[1]

	switch bbc.Request.Database.Type {
	case "oceanbase":
		if con.NodeName != "" && con.ClusterName != "" {
			bbc.Request.Database.Username = fmt.Sprintf("%s@%s#%s", username, con.NodeName, con.ClusterName)
		} else {
			bbc.Request.Database.Username = username
		}
	case "tidb":
		bbc.Request.Database.Type = "mysql"
		bbc.Request.Database.Username = username
	default:
		bbc.Request.Database.Username = username
	}
	//bbc.Request.Database.Username = con.DbUsername
	//var er error
	//bbc.Request.Database.Password, er = encryption.Decode_Nologging(con.DbPassword)
	//if er != nil {
	//	bbc.Request.Database.Password = con.DbPassword
	//}
	p, err := encryption.Decode(password)
	if err == nil && p != "" {
		password = p
	}
	bbc.Request.Database.Password = password
	bbc.Request.Database.Version = con.DbVersion
	if bbc.Request.Database.Type == "db2" {
		if con.SecurityMechanism != "" {
			bbc.Request.Database.SecurityMechanism = con.SecurityMechanism
		}
	}
	bbc.getQueryURL()
	return
}

func (bbc *CommonBBC) getQueryURL() {
	dbType := bbc.Instance.Category[1]
	ip := bbc.Instance.Data.IP
	dbPort := bbc.Instance.Account[1].Content.DbPort
	dbName := bbc.Instance.Account[1].Content.DbName
	dbInstanceName := bbc.Instance.Account[1].Content.DbInstanceName
	encoding := bbc.Instance.Account[1].Content.DbEncoding
	var url string
	switch strings.ToLower(dbType) {
	case "oracle":
		url = fmt.Sprintf("jdbc:oracle:thin:@//%s:%d/%s", ip, dbPort, dbName)
		if bbc.Instance.Account[1].Content.JDBCType == "SID" {
			url = fmt.Sprintf("jdbc:oracle:thin:@%s:%d:%s", ip, dbPort, dbName)
		}
		break
	case "db2":
		//url = fmt.Sprintf("jdbc:db2://%s:%d/%s:blockingConnectionTimeout=30;", ip, dbPort, dbName)
		url = fmt.Sprintf("jdbc:db2://%s:%d/%s", ip, dbPort, dbName)
	case "mysql", "tidb":
		url = fmt.Sprintf("jdbc:mysql://%s:%d/%s%s", ip, dbPort, dbName, encoding)
	case "sqlserver":
		url = fmt.Sprintf("jdbc:sqlserver://%s:%d;", ip, dbPort)
		if dbInstanceName != "" {
			url = fmt.Sprintf("jdbc:sqlserver://%s:%d;instanceName=%s;databaseName=%s", ip, dbPort, dbInstanceName, dbName)
		}
	case "postgres", "postgresql":
		url = fmt.Sprintf("jdbc:postgresql://%s:%d/%s", ip, dbPort, dbName)
		break
	case "informix":
		url = fmt.Sprintf("jdbc:informix-sqli://%s:%d/%s:%s", ip, dbPort, dbName, encoding)
		break
	case "dm":
		url = fmt.Sprintf("jdbc:dm://%s:%d/%s", ip, dbPort, dbName)
		break
	case "gbase":
		url = fmt.Sprintf("jdbc:gbase://%s:%d/%s", ip, dbPort, dbName)
		break
	case "shentong":
		url = fmt.Sprintf("jdbc:oscar://%s:%d/%s", ip, dbPort, dbName)
		break
	case "oceanbase":
		url = fmt.Sprintf("jdbc:oceanbase://%s:%d/%s%s", ip, dbPort, dbName, encoding)
		break
	case "hive":
		url = fmt.Sprintf("jdbc:hive://%s:%d/%s", ip, dbPort, dbName)
	default:
		break
	}
	bbc.Request.Database.URL = url
	return
}

func (bbc *CommonBBC) DbQuery(UseGlobalView, SystemSqlMarked, SystemSqlMarkedContent, DB2SelectWithUr string) (err error) {
	defer func() {
		if pr := recover(); pr != nil {
			//debug.PrintStack()
			err = fmt.Errorf(" db pool query: got panic error %v %v", zap.Any("error", pr), zap.String("stack", string(debug.Stack())))
			return
		}
	}()
	err = bbc.formatViews(UseGlobalView, SystemSqlMarked, SystemSqlMarkedContent, DB2SelectWithUr)
	if err != nil {
		return fmt.Errorf("DB-Query:format views error %v", zap.Error(err))
	}
	url := bbc.Url + "/v1/pool/" + strings.ToLower(bbc.Class)
	//req, _ := json.Marshal(bbc.Request)
	//res, err := http.Post(bbc.Url, "application/json;charset=utf-8", bytes.NewBuffer(req))
	//if err != nil {
	//	bblog.Logger.Error("DB-Query:get java query error", zap.Error(err), zap.String("sql", bbc.Request.Params.SQL))
	//	return err
	//}
	client := newClient()
	if bbc.Timeout > 0 {
		client.Timeout = time.Duration(bbc.Timeout) * time.Second
	} else {
		client.Timeout = 500 * time.Second
	}

	payload, err := json.Marshal(bbc.Request)
	if err != nil {
		return fmt.Errorf("marshal request data error %v  %v", zap.Error(err), zap.Any("request", bbc.Request))
	}
	st := time.Now()
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("new http request error %v", zap.Error(err))
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	defer func() {
		if res != nil && res.Body != nil {
			res.Body.Close()
		}
	}()
	if err != nil {
		err = fmt.Errorf("call connPool service error %v %v", zap.Error(err), zap.Duration("time", time.Since(st)))
		if strings.Contains(strings.ToUpper(err.Error()), "TIMEOUT") {
			err = errors.New("call connPool service timeout")
		}
		return err
	}
	rbody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf(" DB-Query:read query body error %v", zap.Error(err))
	}

	if res.StatusCode == 200 {
		_ = json.Unmarshal(rbody, &bbc.Response)
	} else {
		_ = json.Unmarshal(rbody, &bbc.Error)
		//zap.String("sql", bbc.Request.Params.SQL))
		return errors.New(bbc.Error.Message)
	}

	return nil
}

//func (bbc *CommonBBC) auditQuery() (err error) {
//	bbc.Url = bbc.Url + "/v1/pool/oracle/batch_" + bbc.Class
//	bbc.getQueryData()
//	//if this.Class == "explain" {
//	//	this.Request.Params.Schema = schema
//	//}
//	//this.Request.Params.SqlMap = sqlMap
//	req, _ := json.Marshal(bbc.Request)
//	res, err := http.Post(bbc.Url, "application/json;charset=utf-8", bytes.NewBuffer(req))
//	if err != nil {
//		return err
//	}
//
//	rbody, err := ioutil.ReadAll(res.Body)
//	defer res.Body.Close()
//	if err != nil {
//		bblog.Logger.Error("get java query error", zap.Error(err))
//		return err
//	}
//
//	if res.StatusCode == 200 {
//		_ = json.Unmarshal(rbody, &bbc.Response)
//	} else {
//		_ = json.Unmarshal(rbody, &bbc.Error)
//		bblog.Logger.Error("get java query err result", zap.Any("qre", bbc.Error))
//		return errors.New(bbc.Error.Message)
//	}
//
//	return nil
//}

func (bbc *CommonBBC) analyzeDBProxy(data map[string]string) (err error) {
	defer func() {
		if pr := recover(); pr != nil {
			//debug.PrintStack()
			err = fmt.Errorf("db psql analyze: got panic error %v %v", zap.Any("error", pr), zap.String("stack", string(debug.Stack())))
			return
		}
	}()
	client := newClient()

	bbc.Url = bbc.Url + "/v1/anaylize/doSqlAnaylize"
	payload, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", bbc.Url, bytes.NewBuffer(payload))
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	defer func() {
		if res != nil && res.Body != nil {
			res.Body.Close()
		}
	}()
	if err != nil {
		return fmt.Errorf("get java analyze error %v", zap.Error(err))
	}
	rbody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("get java analyze error %v", zap.Error(err))
	}

	if res.StatusCode == 200 {
		err = json.Unmarshal(rbody, &bbc.Analyze)
		if err != nil {
			return err
		}
	} else {
		err = json.Unmarshal(rbody, &bbc.Error)
		if err != nil {
			return err
		}
		return errors.New(bbc.Error.Message)
	}

	return nil
}

//func (bbc *CommonBBC) xmlParseDBProxy(data string) (sl []string, err error) {
//	client := newClient()
//	sl = make([]string, 0)
//	url := viper.GetString("bbc_server") + "/v1/poll/mybatis"
//	//payload, _ := xml.Marshal(data)err
//	req, err := http.NewRequest("POST", url, strings.NewReader(data))
//	if err != nil {
//		bblog.Logger.Error("parse mybatis xml error when get http request", zap.Error(err))
//		return sl, err
//	}
//	req.Header.Add("Content-Type", "application/xml")
//
//	res, err := client.Do(req)
//	if err != nil {
//		bblog.Logger.Error("parse mybatis xml error get http response", zap.Error(err))
//		return sl, err
//	}
//	rbody, err := ioutil.ReadAll(res.Body)
//	defer res.Body.Close()
//	if err != nil {
//		return sl, err
//	}
//	if res.StatusCode == 200 {
//		err = json.Unmarshal(rbody, &sl)
//		if err != nil {
//			bblog.Logger.Error("parse mybatis xml error format response body", zap.Error(err))
//			return sl, err
//		}
//	} else {
//		err = json.Unmarshal(rbody, &bbc.Error)
//		if err != nil {
//			bblog.Logger.Error("parse mybatis xml error format response error message", zap.Error(err))
//			return sl, err
//		}
//		return sl, errors.New(bbc.Error.Message)
//	}
//
//	return sl, nil
//}

func (bbc *CommonBBC) formatViews(UseGlobalView, SystemSqlMarked, SystemSqlMarkedContent, DB2SelectWithUr string) error {
	ugv := UseGlobalView
	if ugv == "false" {
		bbc.Request.Params.SQL = strings.Replace(bbc.Request.Params.SQL, "{{use_global_view}}", "v", -1)
		bbc.Request.Params.SQL = strings.Replace(bbc.Request.Params.SQL, "{{audit_use_global_view}}", "v", -1)
		reg, err := regexp.Compile("(?U)######.+######")
		if err != nil {
			return err
		}
		bbc.Request.Params.SQL = reg.ReplaceAllString(bbc.Request.Params.SQL, "")
	} else {
		bbc.Request.Params.SQL = strings.Replace(bbc.Request.Params.SQL, "{{use_global_view}}", "gv", -1)
		bbc.Request.Params.SQL = strings.Replace(bbc.Request.Params.SQL, "{{audit_use_global_view}}", "gv", -1)
		bbc.Request.Params.SQL = strings.Replace(bbc.Request.Params.SQL, "######", "", -1)
	}

	if SystemSqlMarked == "true" && SystemSqlMarkedContent != "" {
		bbc.Request.Params.SQL += " " + SystemSqlMarkedContent
	}

	if bbc.Request.Database.Type == "db2" && bbc.Class == "dql" {
		if DB2SelectWithUr == "true" {
			ls := strings.ToLower(bbc.Request.Params.SQL)
			reg, err := regexp.Compile(`.*for\s+update.*`)
			if err != nil {
				return fmt.Errorf("regexp compile error %v", zap.Error(err))
			}
			if reg.MatchString(ls) {
				return nil
			}
			reg, err = regexp.Compile(`.*with\s+ur.*`)
			if err != nil {
				return fmt.Errorf("regexp compile error %v", zap.Error(err))
			}
			if reg.MatchString(ls) {
				return nil
			}
			bbc.Request.Params.SQL += " with ur "
		}
	}

	return nil
}

func (bbc *CommonBBC) dbExec() (err error) {
	defer func() {
		if pr := recover(); pr != nil {
			//debug.PrintStack()
			err = fmt.Errorf("db exec: got panic error %v %v", zap.Any("error", pr), zap.String("stack", string(debug.Stack())))
			return
		}
	}()
	url := bbc.Url + "/v1/executor/" + strings.ToLower(bbc.Class)
	req, _ := json.Marshal(bbc.Request)
	res, err := http.Post(url, "application/json;charset=utf-8", bytes.NewBuffer(req))
	defer func() {
		if res != nil && res.Body != nil {
			res.Body.Close()
		}
	}()
	if err != nil {
		return fmt.Errorf("DB-Exec:get java query error %v %v", zap.Error(err), zap.String("sql", bbc.Request.Params.SQL))
	}
	rbody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("DB-Query:read query body error %v", zap.Error(err))
	}

	if res.StatusCode == 200 {
		if bbc.Class == "EXPLAIN" {
			err = json.Unmarshal(rbody, &bbc.Response)
			if err != nil {
				return fmt.Errorf("DB-Query:unmarshal body error %v", zap.Error(err))
			}
		} else {
			//d := json.NewDecoder(bytes.NewBuffer(rbody))
			//d.UseNumber()
			err = json.Unmarshal(rbody, &bbc.Response)
			if err != nil {
				return fmt.Errorf("DB-Query:unmarshal and decode body error %v", zap.Error(err))
			}
		}
		if bbc.Class == "DQL" {
			bbc.Response.AffectRow = len(bbc.Response.Data)
		}
	} else {
		_ = json.Unmarshal(rbody, &bbc.Error)
		//zap.String("sql", bbc.Request.Params.SQL))
		return fmt.Errorf("DB-Exec:get java query err result %v", zap.Any("qre", bbc.Error))
	}
	return nil
}

func (bbc *CommonBBC) dbPageSelect() (err error) {
	defer func() {
		if pr := recover(); pr != nil {
			//debug.PrintStack()
			err = fmt.Errorf("db page select: got panic error %v %v", zap.Any("error", pr), zap.String("stack", string(debug.Stack())))
			return
		}
	}()
	url := bbc.Url + "/v1/executor/" + strings.ToLower(bbc.Class) + "/page"
	req, _ := json.Marshal(bbc.Request)
	res, err := http.Post(url, "application/json;charset=utf-8", bytes.NewBuffer(req))
	defer func() {
		if res != nil && res.Body != nil {
			res.Body.Close()
		}
	}()
	if err != nil {
		return fmt.Errorf("DB-Exec:get java query error %v %v", zap.Error(err), zap.String("sql", bbc.Request.Params.SQL))
	}
	rbody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("DB-Query:read query body error %v", zap.Error(err))
	}

	if res.StatusCode == 200 {
		if bbc.Class == "EXPLAIN" {
			err = json.Unmarshal(rbody, &bbc.Response)
			if err != nil {
				return fmt.Errorf("DB-Query:unmarshal body error %v", zap.Error(err))
			}
		} else {
			//d := json.NewDecoder(bytes.NewBuffer(rbody))
			//d.UseNumber()
			//err = d.Decode(&bbc.Response)
			err = json.Unmarshal(rbody, &bbc.Response)
			if err != nil {
				return fmt.Errorf("DB-Query:unmarshal and decode body error %v", zap.Error(err))
			}
		}
		if bbc.Class == "DQL" {
			bbc.Response.AffectRow = len(bbc.Response.Data)
		}
	} else {
		_ = json.Unmarshal(rbody, &bbc.Error)
		//zap.String("sql", bbc.Request.Params.SQL))
		return fmt.Errorf(bbc.Error.Message)
	}
	return nil
}

func CatchErrorCode(dbType string, qre QueryResultErr) (code string) {
	switch dbType {
	case "oracle":
		return catchOracleErrorCode(qre.Message)
	case "mysql":
		return catchMysqlErrorCode(qre.Message)
	case "dm":
		return catchDMErrorCode(qre.Message)
	default:
		return ErrUnDefineError
	}
	return
}

func catchOracleErrorCode(message string) (code string) {
	if strings.Contains(message, "IO Error: The Network Adapter could not establish the connection") {
		return ErrDBConnectFailed
	}
	reg, _ := regexp.Compile(`ORA-\d{5}`)
	sl := reg.FindAllString(message, 1)
	if len(sl) > 0 {
		_, ok := OracleCodeMap[sl[0]]
		if ok {
			code = OracleCodeMap[sl[0]]
		} else {
			code = ErrUnDefineError
		}
	} else {
		code = ErrUnDefineError
	}
	return
}

func catchMysqlErrorCode(message string) (code string) {
	if strings.Contains(message, "Access denied for user") {
		return ErrDBUserAuth
	} else {
		code = ErrDBConnectTimeout
	}
	return
}

func catchDMErrorCode(message string) (code string) {
	if strings.Contains(message, "Invalid username or password") {
		return ErrDBUserAuth
	} else {
		code = ErrDBConnectTimeout
	}
	return
}

type connResultOK struct {
	ID                   string         `json:"id"`
	Key                  string         `json:"key"`
	ConnectionCreateTime int            `json:"connectionCreateTime"`
	Connection30Record   []int          `json:"connection30Record"`
	Call15Record         []int          `json:"call15Record"`
	Prepared15Record     []int          `json:"prepared15Record"`
	Settings             datatypes.JSON `json:"settings"`
}

type proxyResultErr struct {
	StatusCode int    `json:"status_code"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	Detail     string `json:"detail"`
}

func (bbc *CommonBBC) GetDBConnections() (status int, pro []connResultOK, pre proxyResultErr, err error) {
	client := newClient()
	url := bbc.Url + "/v1/connections"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", "application/json")
	res, err := client.Do(req)
	defer func() {
		if res.Body != nil {
			res.Body.Close()
		}
	}()
	if err != nil {
		return 0, pro, pre, err
	}
	rbody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return 0, pro, pre, err
	}
	if res.StatusCode == 200 {
		_ = json.Unmarshal(rbody, &pro)
	} else {
		_ = json.Unmarshal(rbody, &pre)
	}
	return res.StatusCode, pro, pre, nil
}

type proxyConnRequest struct {
	RequestID string `json:"request_id"`
}

func (bbc *CommonBBC) ManageDBConnections(typ string) (err error) {
	data := proxyConnRequest{
		RequestID: bbc.Request.RequestID,
	}
	//bblog.Logger.Debug("DeleteConn request info.", zap.Any("request", data))
	client := newClient()
	url := bbc.Url + "/v1/connections/" + bbc.Request.Database.Key
	if typ == "cancel" {
		url += "/cancel"
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal request error %v", zap.Error(err))
	}
	req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("new request error %v", zap.Error(err))
	}
	req.Header.Add("Content-Type", "application/json")
	res, err := client.Do(req)
	defer func() {
		if res != nil && res.Body != nil {
			res.Body.Close()
		}
	}()
	if err != nil {
		return fmt.Errorf("do request error %v", zap.Error(err))
	}
	rbody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	//bblog.Logger.Debug(typ+" db proxy connections response info.", zap.String("response", string(rbody)))
	if res.StatusCode != 200 {
		_ = json.Unmarshal(rbody, &bbc.Error)
	}
	return nil
}

type statsConnResponse struct {
	Total int `json:"total"`
}

func (bbc *CommonBBC) StatsDBProxyConn() (status int, pro statsConnResponse, pre proxyResultErr, err error) {
	client := newClient()
	url := bbc.Url + "/v1/stats/connections"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", "application/json")
	res, _ := client.Do(req)
	defer func() {
		if res.Body != nil {
			res.Body.Close()
		}
	}()
	rbody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return 0, pro, pre, err
	}
	//bblog.Logger.Debug("statsDBProxyConn response info.", zap.String("response", string(rbody)))
	if res.StatusCode == 200 {
		_ = json.Unmarshal(rbody, &pro)
	} else {
		_ = json.Unmarshal(rbody, &pre)
	}
	return res.StatusCode, pro, pre, nil
}

func (bbc *CommonBBC) CleanData() {
	bbc.Response = QueryResultOK{}
	bbc.Error = QueryResultErr{}
	bbc.Analyze = []analyzeResult{}
}

type SQLExecuteInfo struct {
	SQLID         string
	CPUTime       float64
	LogicalRead   float64
	PhysicalRead  float64
	PlanHashValue string
}

func (bbc *CommonBBC) GetSQLExecuteInfo(last SQLExecuteInfo) (info SQLExecuteInfo, err error) {
	s := ""
	//switch bbc.DBConnection.DbType {
	//case "oracle":
	s = `select  stat.name,ses.value   
   from v$sesstat ses , v$statname stat
   where stat.statistic# = ses.statistic# and
   ses.sid=(select sid from v$mystat where rownum <2 )
   and stat.name  in ('session logical reads','physical read total IO requests','CPU used by this session')
union 
  select sql_id name ,plan_hash_value value from v$sql where sql_id = (select prev_sql_id from v$session where sid=(select sid from v$mystat where rownum=1))`

	//default:
	//	return
	//
	//}
	//bbc.init()
	bbc.Request.Params.SQL = s
	bbc.Class = "dql"
	err = bbc.dbExec()
	if err != nil {
		return
	}
	if bbc.Error.Message != "" {
		err = errors.New(bbc.Error.Message)
		return
	}
	if len(bbc.Response.Data) > 0 {
		for _, d := range bbc.Response.Data {
			name := fmt.Sprintf("%v", d["NAME"])
			if name == "" {
				continue
			}
			value := d["VALUE"]
			switch name {
			case "session logical reads":
				num, err := cast.ToFloat64E(value)
				if err != nil {
					//bblog.Logger.Error("covert string to float64 error", zap.Error(err))
					continue
				}
				info.LogicalRead = num - last.LogicalRead
			case "physical read total IO requests":
				num, err := cast.ToFloat64E(value)
				if err != nil {
					//bblog.Logger.Error("covert string to float64 error", zap.Error(err))
					continue
				}
				info.PhysicalRead = num - last.PhysicalRead
			case "CPU used by this session":
				num, err := cast.ToFloat64E(value)
				if err != nil {
					//bblog.Logger.Error("covert string to float64 error", zap.Error(err))
					continue
				}
				info.CPUTime = num - last.CPUTime
			default:
				info.SQLID = name
				info.PlanHashValue = cast.ToString(value)
			}
		}
	}
	return
}
