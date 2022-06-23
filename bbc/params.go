package bbc

type QueryRequest struct {
	RequestID string `json:"request_id"`
	Database  dbInfo `json:"database"`
	Params    params `json:"params"`
}

type dbInfo struct {
	Username          string `json:"username"`
	Password          string `json:"password"`
	Version           string `json:"version"`
	Type              string `json:"type"`
	URL               string `json:"url"`
	Key               string `json:"key"`
	SecurityMechanism string `json:"securityMechanism"`
	Did               int    `json:"did"`
	AutoCommit        bool   `json:"auto_commit"`
}

type params struct {
	SQL     string            `json:"sql"`
	Schema  string            `json:"schema"`
	DBName  string            `json:"db_name"`
	SqlMap  map[string]string `json:"sqlMap"`
	Execute bool              `json:"execute"`
	Limit   int               `json:"limit"`
	Type    string            `json:"type"`
	StartNo int               `json:"startNo"`
}
type QueryResultOK struct {
	TableNames     []string                 `json:"table_names"`
	AffectRow      int                      `json:"affect_row"`
	Duration       int                      `json:"duration"`
	Context        string                   `json:"context"`
	Data           []map[string]interface{} `json:"data"`
	ColumnType     map[string]string        `json:"column_type"`
	ColumnSequence []string                 `json:"column_sequence"`
	EOF            bool                     `json:"eof"`
}

type QueryResultErr struct {
	StatusCode int    `json:"status_code"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	Detail     string `json:"detail"`
}

type Instance struct {
	ID             int               `json:"id"`
	Category       []string          `json:"category"`
	Usefor         []string          `json:"usefor"`
	Account        []InstanceAccount `json:"account"`
	Data           InstanceData      `json:"data"`
	GroupID        int               `json:"group_id"`
	CpuCount       int               `json:"cpu_count"`
	Scene          string            `json:"scene"`
	InstanceStatus string            `json:"instance_status"`
	SLA            string            `json:"sla"`
	ServerID       int               `json:"server_id"`
}
type analyzeResult struct {
	TableInfos []tableInfo `json:"table_infos"`
	Conditions []condition `json:"conditions"`
	Columns    []column    `json:"columns"`
	Functions  []string    `json:"functions"`
	SQL        string      `json:"sql"`
	FormatSQL  string      `json:"format_sql"`
}

type tableInfo struct {
	TableName string `json:"table_name"`
	Type      string `json:"type"`
}

type condition struct {
	Operator   string        `json:"operator"`
	ColumnName string        `json:"column_name"`
	Values     []interface{} `json:"values"`
	Condition  string        `json:"condition"`
}

type column struct {
	Owner      string `json:"owner"`
	TableName  string `json:"table_name"`
	Column     string `json:"column"`
	ColumnType string `json:"column_type"`
}

type InstanceAccount struct {
	ID      int             `json:"id"`
	Content InstanceContent `json:"content"`
}

type InstanceContent struct {
	DbType         string `json:"db_type"`
	DbPort         int    `json:"db_port"`
	DbName         string `json:"db_name"`
	DbUsername     string `json:"db_username"`
	DbPassword     string `json:"db_password"`
	DbVersion      string `json:"db_version"`
	DbInstanceName string `json:"db_instance_name"`
	DbEncoding     string `json:"db_encoding"`
	ClusterName    string `json:"cluster_name"`
	NodeName       string `json:"node_name"`
	NodeType       string `json:"node_type"`

	SshUsername string `json:"ssh_username"`
	SshPassword string `json:"ssh_password"`
	SshPort     int    `json:"ssh_port"`
	SshKey      string `json:"ssh_key"`

	SnmpPort    uint16 `json:"snmp_port"`
	SnmpVersion int    `json:"snmp_version"`
	Community   string `json:"snmp_public"`

	Db2Profile        string `json:"db2_profile"`
	ExplainSchema     string `json:"explain_schema"`
	SecurityMechanism string `json:"securityMechanism"`
	JDBCType          string `json:"jdbc_type"`
	SshEnable         bool   `json:"ssh_enable"`
}

type InstanceData struct {
	IP            string `json:"ip"`
	Name          string `json:"name"`
	MonitorStatus bool   `json:"monitor_status"`
	AuditStatus   bool   `json:"audit_status"`
	Method        string `json:"method"`
	Env           string `json:"env"`
	Version       string `json:"version"`
}

const (
	ErrUnDefineError    = "ErrUnDefineError"
	ErrDBConnectFailed  = "ErrDBConnectFailed"
	ErrDBUserAuth       = "ErrDBUserAuth"
	ErrDBConnectTimeout = "ErrDBConnectTimeout"
)

var OracleCodeMap = map[string]string{
	"ORA-12514": "ErrDBServiceName",
	"ORA-01045": "ErrDBUserPrivilege",
	"ORA-01017": "ErrDBUserAuth",
	"ORA-28000": "ErrDBUserLocked",
	"ORA-00257": "ErrDBArchiver",
}
