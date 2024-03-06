package typeconv

import (
	"fmt"
	"strings"

	"github.com/cockroachdb/cockroachdb-parser/pkg/geo/geopb"
	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/types"
)

// This is based on managed-service/migration/internal/mysql/types.go.
func (cm *MySQLColumnMeta) ToDefaultCRDBType(
	dataType, colName string,
) (*types.T, []*TypeConvError) {
	tfs := make([]*TypeConvError, 0)

	makeFailure := func(t string, defToString bool) *TypeConvError {
		defStr := ""
		if defToString {
			defStr = " - defaulting to string"
		}
		return &TypeConvError{
			ShortDescription: UnsupportedColumnType(t),
			Message: fmt.Sprintf(
				"type %s from column %s is unsupported%s",
				t,
				colName,
				defStr,
			),
			Blocking: true,
		}
	}

	if cm.CollationName != "" {
		tfs = append(
			tfs,
			&TypeConvError{
				ShortDescription: UnsupportedCollate(cm.CollationName),
				Message:          fmt.Sprintf("collate %s was not added", cm.CollationName),
			},
		)
	}

	switch t := strings.ToLower(dataType); t {
	// *parser.StringDataTypeContext
	case "char", "character", "varchar", "nchar", "nvarchar":
		if cm.NumPrecision != -1 {
			return types.MakeVarChar(cm.NumPrecision), tfs
		}
		return types.VarChar, tfs
	case "tinytext", "text", "mediumtext", "longtext":
		return types.String, tfs
	// *parser.SpatialDataTypeContext
	case "geometry":
		return types.Geometry, tfs
	case "linestring":
		return types.MakeGeometry(geopb.ShapeType_LineString, 0), tfs
	case "point":
		return types.MakeGeometry(geopb.ShapeType_Point, 0), tfs
	case "polygon":
		return types.MakeGeometry(geopb.ShapeType_Polygon, 0), tfs
	case "multipoint":
		return types.MakeGeometry(geopb.ShapeType_MultiPoint, 0), tfs
	case "multilinestring":
		return types.MakeGeometry(geopb.ShapeType_MultiLineString, 0), tfs
	case "multipolygon":
		return types.MakeGeometry(geopb.ShapeType_MultiPolygon, 0), tfs
	case "geometrycollection", "geomcollection":
		return types.MakeGeometry(geopb.ShapeType_GeometryCollection, 0), tfs
	case "json":
		return types.Jsonb, tfs
	// *parser.DimensionDataTypeContext
	case "tinyint", "int1":
		return types.Int2, append(tfs, &TypeConvError{
			ShortDescription: UnsupportedTinyInt,
			Message: fmt.Sprintf(
				"column %s uses tinyint, which does not exist in CockroachDB - this has been promoted to INT2",
				colName,
			),
		})
	case "blob":
		return types.Bytes, tfs
	case "smallint", "int2":
		return types.Int2, tfs
	case "mediumint", "int", "integer", "int4":
		return types.Int4, tfs
	case "bigint", "int8":
		return types.Int, tfs
	case "float":
		return types.Float4, tfs
	case "double":
		return types.Float, tfs
	case "decimal", "numeric", "real":
		if cm.NumScale != -1 || cm.NumPrecision != -1 {
			if cm.NumScale > cm.NumPrecision {
				tfs = append(tfs, &TypeConvError{
					ShortDescription: InvalidDecimalArgs,
					Message:          fmt.Sprintf("precision %d cannot be bigger than cm.NumScale %d for column %s", cm.NumPrecision, cm.NumScale, colName),
					Blocking:         true,
				})
				return types.Decimal, tfs
			}
			return types.MakeDecimal(cm.NumPrecision, cm.NumScale), tfs
		}
		return types.Decimal, tfs
	case "binary", "varbinary":
		if cm.NumPrecision != -1 {
			tfs = append(tfs, &TypeConvError{
				ShortDescription: UnsupportedBytesMax,
				Message:          fmt.Sprintf("column %s specifies a max length, which is unsupported in CockroachDB", colName),
			})
		}
		return types.Bytes, tfs
	case "datetime":
		if cm.DatetimePrecision != -1 {
			return types.MakeTimestamp(cm.DatetimePrecision), tfs
		}
		return types.MakeTimestamp(0), tfs
	case "timestamp":
		if cm.DatetimePrecision != -1 {
			return types.MakeTimestampTZ(cm.DatetimePrecision), tfs
		}
		return types.MakeTimestampTZ(0), tfs
	case "time":
		if cm.DatetimePrecision != -1 {
			return types.MakeTime(cm.DatetimePrecision), tfs
		}
		return types.Time, tfs
	case "bit":
		return types.VarBit, tfs
	// *parser.SimpleDataTypeContext
	case "date":
		return types.Date, tfs
	case "tinyblob", "mediumblob", "longblob":
		return types.Bytes, tfs
	case "bool", "boolean":
		return types.Bool, tfs
	case "enum":
		return types.AnyEnum, tfs
	default:
		tfs = append(tfs, makeFailure(t, false))
		return types.Unknown, tfs
	}
}

// MySQLColumnMeta collects the information about the column in a MySQL table. This information are stored in the
// information_schema
type MySQLColumnMeta struct {
	ColumnDefault     string
	CharMaxLen        int
	CharOctetLen      int
	NumPrecision      int32
	NumScale          int32
	DatetimePrecision int32
	CharSetName       string
	CollationName     string
	ColumnType        string
	ColumnKey         string
	Extra             string
}

// NewMySQLColumnMeta returns a new MySQLColumnMeta objects with unset/default value for all fields.
func NewMySQLColumnMeta() *MySQLColumnMeta {
	return &MySQLColumnMeta{
		NumPrecision:      -1,
		NumScale:          -1,
		DatetimePrecision: -1,
		CharMaxLen:        -1,
		CharOctetLen:      -1,
	}
}
