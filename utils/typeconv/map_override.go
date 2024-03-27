package typeconv

type OverrideTypeMap struct {
	ColumnTypeMaps []ColumnTypeMap `json:"column-type-maps"`
}

type ColumnTypeMap struct {
	ColumnName string   `json:"column"`
	TypeKVs    []TypeKV `json:"type-kvs"`
}

type TypeKV struct {
	SourceType string `json:"source-type"`
	CRDBType   string `json:"crdb-type"`
}
