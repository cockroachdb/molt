package mysqlconv

import (
	"strings"
	"time"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/cockroachdb/cockroachdb-parser/pkg/util/duration"
	"github.com/cockroachdb/cockroachdb-parser/pkg/util/json"
	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/parsectx"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/lib/pq/oid"
)

func ConvertRowValue(typMap *pgtype.Map, val []byte, typOID oid.Oid) (tree.Datum, error) {
	if val == nil {
		return tree.DNull, nil
	}
	switch typOID {
	case pgtype.VarcharOID, pgtype.TextOID:
		return tree.NewDString(string(val)), nil
	case pgtype.Float4OID, pgtype.Float8OID:
		return tree.ParseDFloat(string(val))
	case pgtype.Int2OID, pgtype.Int4OID, pgtype.Int8OID:
		return tree.ParseDInt(string(val))
	case pgtype.JSONOID, pgtype.JSONBOID:
		j, err := json.MakeJSON(string(val))
		if err != nil {
			return nil, errors.Wrapf(err, "error decoding json for %v", val)
		}
		return tree.NewDJSON(j), nil
	case pgtype.TimestampOID:
		v := string(val)
		if strings.HasPrefix(v, "0000-") {
			return tree.DNull, nil
		}
		ret, _, err := tree.ParseDTimestamp(parsectx.ParseContext, v, time.Microsecond)
		return ret, err
	case pgtype.TimestamptzOID:
		v := string(val)
		ret, _, err := tree.ParseDTimestampTZ(parsectx.ParseContext, v, time.Microsecond)
		return ret, err
	case pgtype.TimeOID:
		// https://dev.mysql.com/doc/refman/8.0/en/time.html
		// https://dev.mysql.com/doc/refman/8.0/en/date-and-time-literals.html#:~:text=MySQL%20recognizes%20TIME%20values%20in%20these%20formats%3A
		// From MySQL's doc:
		// MySQL retrieves and displays TIME values in 'hh:mm:ss' format (or 'hhh:mm:ss' format for large hours values).
		// TIME values may range from '-838:59:59' to '838:59:59'. The hours part may be so large because the TIME type
		// can be used not only to represent a time of day (which must be less than 24 hours), but also elapsed time or
		// a time interval between two events (which may be much greater than 24 hours, or even negative).
		v := string(val)
		var ret tree.Datum
		var err error
		ret, _, err = tree.ParseDTime(parsectx.ParseContext, v, time.Microsecond)
		if err == nil {
			return ret, nil
		}
		// If the value cannot be parsed a time datum, try parse it as an interval. See explanation above.
		ret, err = tree.ParseDInterval(duration.IntervalStyle_SQL_STANDARD, v)
		if err != nil {
			return ret, errors.Wrapf(err, "input %q cannot be parsed as time or interval", v)
		}
		return ret, nil
	case pgtype.DateOID:
		ret, _, err := tree.ParseDDate(parsectx.ParseContext, string(val))
		return ret, err
	case pgtype.ByteaOID:
		return tree.NewDBytes(tree.DBytes(val)), nil
	case pgtype.NumericOID:
		return tree.ParseDDecimal(string(val))
	case pgtype.BitOID, pgtype.VarbitOID:
		return tree.ParseDBitArray(string(val))
	case oid.T_anyenum:
		return tree.NewDString(string(val)), nil
	}
	return nil, errors.AssertionFailedf("value type OID %d not yet translatable", typOID)
}

func ConvertRowValues(typMap *pgtype.Map, vals [][]byte, typOIDs []oid.Oid) (tree.Datums, error) {
	ret := make(tree.Datums, len(vals))
	if len(vals) != len(typOIDs) {
		return nil, errors.AssertionFailedf("val length != oid length: %v vs %v", vals, typOIDs)
	}
	for i := range vals {
		var err error
		if ret[i], err = ConvertRowValue(typMap, vals[i], typOIDs[i]); err != nil {
			return nil, err
		}
	}
	return ret, nil
}
