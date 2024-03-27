package typeconv

import (
	"encoding/json"
	"os"

	"github.com/cockroachdb/errors"
)

// OverrideTypeMap correspond to json of the following format:
// [
//
//	{
//	  "column": "age",
//	  "type-kvs": [
//	    {
//	      "source-type": "int",
//	      "crdb-type": "INT"
//	    },
//	    {
//	      "source-type": "float",
//	      "crdb-type": "FLOAT"
//	    }
//	  ]
//	},
//	{
//	  "column": "name",
//	  "type-kvs": [
//	    {
//	      "source-type": "string",
//	      "crdb-type": "TEXT"
//	    }
//	  ]
//	}
//
// ]
type OverrideTypeMap []ColumnTypeMap

func (m *OverrideTypeMap) String() string {
	jsonData, err := json.Marshal(m)
	// This should not happen but just in case.
	if err != nil {
		panic(errors.Wrap(err, "cannot get the json representation of the type override map"))
	}
	return string(jsonData)
}

type ColumnTypeMap struct {
	ColumnName string   `json:"column"`
	TypeKVs    []TypeKV `json:"type-kvs"`
}

type TypeKV struct {
	SourceType string `json:"source-type"`
	CRDBType   string `json:"crdb-type"`
}

func OverrideTypeMapFromFile(filepath string) (*OverrideTypeMap, error) {
	var res = &OverrideTypeMap{}
	bytesValus, err := os.ReadFile(filepath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read json file %s for type map", filepath)
	}
	if err := json.Unmarshal(bytesValus, res); err != nil {
		return nil, err
	}
	return res, nil
}
