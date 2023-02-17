package verification

import (
	"time"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/cockroachdb/cockroachdb-parser/pkg/util/duration"
	"github.com/cockroachdb/cockroachdb-parser/pkg/util/json"
	"github.com/cockroachdb/cockroachdb-parser/pkg/util/timeofday"
	"github.com/cockroachdb/cockroachdb-parser/pkg/util/timeutil/pgdate"
	"github.com/cockroachdb/cockroachdb-parser/pkg/util/uuid"
	"github.com/cockroachdb/errors"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/lib/pq/oid"
)

type parseTimeContext struct{}

var _ tree.ParseTimeContext = (*parseTimeContext)(nil)

func (p parseTimeContext) GetRelativeParseTime() time.Time {
	return time.Now().UTC()
}

func (p parseTimeContext) GetIntervalStyle() duration.IntervalStyle {
	return duration.IntervalStyle_POSTGRES
}

func (p parseTimeContext) GetDateStyle() pgdate.DateStyle {
	return pgdate.DefaultDateStyle()
}

var timeCtx = &parseTimeContext{}

func convertRowValue(val any, typOID OID) (tree.Datum, error) {
	switch typOID {
	case pgtype.BoolOID:
		return tree.MakeDBool(tree.DBool(val.(bool))), nil
	case pgtype.QCharOID, pgtype.VarcharOID, pgtype.TextOID:
		return tree.NewDString(val.(string)), nil
	case pgtype.NameOID:
		return tree.NewDName(val.(string)), nil
	case pgtype.Float4OID:
		return tree.NewDFloat(tree.DFloat(val.(float32))), nil
	case pgtype.Float8OID:
		return tree.NewDFloat(tree.DFloat(val.(float64))), nil
	case pgtype.Int2OID:
		return tree.NewDInt(tree.DInt(val.(int16))), nil
	case pgtype.Int4OID:
		return tree.NewDInt(tree.DInt(val.(int32))), nil
	case pgtype.Int8OID:
		return tree.NewDInt(tree.DInt(val.(int64))), nil
	case pgtype.OIDOID:
		return tree.NewDOid(oid.Oid(val.(uint32))), nil
	case pgtype.JSONOID, pgtype.JSONBOID:
		j, err := json.MakeJSON(val)
		if err != nil {
			return nil, errors.Wrapf(err, "error decoding json for %v", val)
		}
		return tree.NewDJSON(j), nil
	case pgtype.UUIDOID:
		u, err := uuid.FromString(val.(string))
		if err != nil {
			return nil, errors.Wrapf(err, "error decoding UUID %v", val)
		}
		return tree.NewDUuid(tree.DUuid{UUID: u}), nil
	case pgtype.TimestampOID:
		return tree.MakeDTimestamp(val.(time.Time), time.Microsecond)
	case pgtype.TimestamptzOID:
		return tree.MakeDTimestampTZ(val.(time.Time).UTC(), time.Microsecond)
	case pgtype.TimeOID:
		return tree.MakeDTime(timeofday.FromInt(val.(pgtype.Time).Microseconds)), nil
	case pgtype.DateOID:
		d, err := pgdate.MakeDateFromTime(val.(time.Time))
		if err != nil {
			return nil, errors.Wrapf(err, "error converting date %v", val)
		}
		return tree.NewDDate(d), nil
	case pgtype.ByteaOID:
		return tree.NewDBytes(tree.DBytes(val.([]byte))), nil
	case OID(oid.T_timetz): // does not exist in pgtype.
		d, _, err := tree.ParseDTimeTZ(timeCtx, val.(string), time.Microsecond)
		return d, err
	}
	return nil, errors.AssertionFailedf("value %v (%T) of type OID %d not yet translatable", val, val, typOID)
}

func convertRowValues(vals []any, typOIDs []OID) (tree.Datums, error) {
	ret := make(tree.Datums, len(vals))
	if len(vals) != len(typOIDs) {
		return nil, errors.AssertionFailedf("val length != oid length: %v vs %v", vals, typOIDs)
	}
	for i := range vals {
		var err error
		if ret[i], err = convertRowValue(vals[i], typOIDs[i]); err != nil {
			return nil, err
		}
	}
	return ret, nil
}