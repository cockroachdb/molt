package typeconv

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/parser"
	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/types"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog"
)

// columnTypeMapsJson correspond to json of the following format:
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
type columnTypeMapsJson []*ColumnTypeMapJson

// Json fields must be exported so that json can be unmarshalled.
type ColumnTypeMapJson struct {
	ColumnName  string        `json:"column"`
	TypeKVsJson []*TypeKVJson `json:"type-kvs"`
}

type TypeKVJson struct {
	SourceType string `json:"source-type"`
	CrdbType   string `json:"crdb-type"`
}

func (m columnTypeMapsJson) String() string {
	res, err := json.Marshal(m)
	// This should not happen.
	if err != nil {
		panic(err)
	}
	return string(res)
}

// ColumnTypeMap: column name -> {source type: crdb type}.
type ColumnTypeMap map[string]TypeKV

// TypeKV: source type -> crdb type.
type TypeKV map[string]*CRDBType

// CRDBType is defined so that we can redefine the string representation.
type CRDBType types.T

func (t CRDBType) String() string {
	tmp := types.T(t)
	return fmt.Sprintf("{%s}", strings.TrimSuffix(tmp.DebugString(), " "))
}

// toColumnTypeMap is to converted the marshalled "json" struct to the map struct.
func (ms columnTypeMapsJson) toColumnTypeMap() (ColumnTypeMap, error) {
	res := make(ColumnTypeMap)
	for _, m := range ms {
		res[m.ColumnName] = make(TypeKV)
		for _, kv := range m.TypeKVsJson {
			crdbT, err := getTypeFromName(strings.ToLower(kv.CrdbType))
			if err != nil {
				return nil, errors.Newf("cannot get the crdb type for %s", kv.CrdbType)
			}
			crdbTyp := CRDBType(*crdbT)
			res[m.ColumnName][kv.SourceType] = &crdbTyp
		}
	}
	return res, nil
}

func GetOverrideTypeMapFromFile(filepath string, logger zerolog.Logger) (ColumnTypeMap, error) {
	var jsonRes = columnTypeMapsJson{}
	bytesValus, err := os.ReadFile(filepath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read json file %s for type map", filepath)
	}
	if err := json.Unmarshal(bytesValus, &jsonRes); err != nil {
		return nil, err
	}
	logger.Debug().Msgf("received type map: %s", jsonRes.String())

	res, err := jsonRes.toColumnTypeMap()
	if err != nil {
		return nil, err
	}
	logger.Debug().Msgf("converted type map: %s", res)
	return res, nil
}

func getTypeFromName(typ string) (*types.T, error) {
	stmt, err := parser.Parse(fmt.Sprintf("SELECT 1::%s", typ))
	if err != nil {
		return nil, err
	}

	ast := stmt[0].AST.(*tree.Select)
	selectCaluse := ast.Select.(*tree.SelectClause)
	castExpr := selectCaluse.Exprs[0].Expr.(*tree.CastExpr)
	res := castExpr.Type.(*types.T)

	return res, nil
}
