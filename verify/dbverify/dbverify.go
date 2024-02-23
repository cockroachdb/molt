// Package dbverify is responsible for verifying two different databases match.
package dbverify

import (
	"context"
	"sort"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/dbtable"
	"github.com/cockroachdb/molt/utils"
)

type connWithTables struct {
	dbconn.Conn
	tableMetadata []dbtable.DBTable
}

type tableVerificationIterator struct {
	tables  connWithTables
	currIdx int
}

func (c *tableVerificationIterator) done() bool {
	return c.currIdx >= len(c.tables.tableMetadata)
}

func (c *tableVerificationIterator) next() {
	c.currIdx++
}

func (c *tableVerificationIterator) curr() dbtable.DBTable {
	return c.tables.tableMetadata[c.currIdx]
}

// Verify verifies tables exist in all databases.
func Verify(ctx context.Context, conns dbconn.OrderedConns) (utils.Result, error) {
	// Grab all tables and verify them.
	var in []connWithTables
	for _, conn := range conns {
		var tms []dbtable.DBTable
		switch conn := conn.(type) {
		case *dbconn.MySQLConn:
			rows, err := conn.QueryContext(
				ctx,
				`SELECT table_name FROM information_schema.tables
WHERE table_schema = database() AND table_type = "BASE TABLE"
ORDER BY table_name`,
			)
			if err != nil {
				return utils.Result{}, err
			}

			for rows.Next() {
				var tn string
				if err := rows.Scan(&tn); err != nil {
					return utils.Result{}, errors.Wrap(err, "error decoding tables metadata")
				}
				// Fake the public schema for now.
				tm := dbtable.DBTable{
					Name: dbtable.Name{
						Schema: "public",
						Table:  tree.Name(tn),
					},
				}
				tms = append(tms, tm)
			}
			if rows.Err() != nil {
				return utils.Result{}, errors.Wrap(err, "error collecting tables metadata")
			}
			if err := rows.Close(); err != nil {
				return utils.Result{}, err
			}
		case *dbconn.OracleConn:
			rows, err := conn.QueryContext(
				ctx,
				`select tablespace_name, table_name from all_tables where tablespace_name not in ( 'SYSTEM', 'SYSAUX', 'ADMINISTRATOR' )`,
			)
			if err != nil {
				return utils.Result{}, err
			}

			for rows.Next() {
				var sn string
				var tn string
				if err := rows.Scan(&sn, &tn); err != nil {
					return utils.Result{}, errors.Wrap(err, "error decoding tables metadata")
				}
				tm := dbtable.DBTable{
					Name: dbtable.Name{
						Schema: tree.Name(sn),
						Table:  tree.Name(tn),
					},
				}
				tms = append(tms, tm)
			}
			if rows.Err() != nil {
				return utils.Result{}, errors.Wrap(err, "error collecting tables metadata")
			}
			if err := rows.Close(); err != nil {
				return utils.Result{}, err
			}
		case *dbconn.PGConn:
			rows, err := conn.Query(
				ctx,
				`SELECT pg_class.oid, pg_class.relname, pg_namespace.nspname
FROM pg_class
JOIN pg_namespace on (pg_class.relnamespace = pg_namespace.oid)
WHERE relkind = 'r' AND pg_namespace.nspname NOT IN ('pg_catalog', 'information_schema', 'crdb_internal', 'pg_extension')
ORDER BY 3, 2`,
			)
			if err != nil {
				return utils.Result{}, err
			}

			for rows.Next() {
				var tm dbtable.DBTable
				if err := rows.Scan(&tm.OID, &tm.Table, &tm.Schema); err != nil {
					return utils.Result{}, errors.Wrap(err, "error decoding tables metadata")
				}
				tms = append(tms, tm)
			}
			if rows.Err() != nil {
				return utils.Result{}, errors.Wrap(err, "error collecting tables metadata")
			}
			rows.Close()
		default:
			return utils.Result{}, errors.Newf("connection %T not supported", conn)
		}

		// Sort tables by schemas and names.
		sort.Slice(tms, func(i, j int) bool {
			return tms[i].Less(tms[j])
		})
		in = append(in, connWithTables{
			Conn:          conn,
			tableMetadata: tms,
		})
	}

	var iterators [2]tableVerificationIterator
	for i := range in {
		iterators[i] = tableVerificationIterator{
			tables: in[i],
		}
	}
	return compare(iterators), nil
}

// compare compares two lists of tables.
// It assumes tables are in sorted order in each iterator.
func compare(iterators [2]tableVerificationIterator) utils.Result {
	ret := utils.Result{}
	// Iterate through all tables in source of truthIterator, moving iterators
	// across
	truthIterator := &iterators[0]
	nonTruthIterator := &iterators[1]
	for !truthIterator.done() {
		// If the iterator is done, that means we are missing tables
		// from the truth value. Mark nonTruthIterator as 1 to signify nonTruthIterator as a missing
		// tables.
		compareVal := 1
		if !nonTruthIterator.done() {
			compareVal = nonTruthIterator.curr().Compare(truthIterator.curr())
		}
		switch compareVal {
		case -1:
			// Extraneous row compared to source of truthIterator.
			ret.ExtraneousTables = append(
				ret.ExtraneousTables,
				utils.ExtraneousTable{DBTable: nonTruthIterator.curr()},
			)
			nonTruthIterator.next()
		case 0:
			var tables [2]dbtable.DBTable
			for i := range iterators {
				tables[i] = iterators[i].curr()
			}
			ret.Verified = append(ret.Verified, tables)
			nonTruthIterator.next()
			truthIterator.next()
		case 1:
			// Missing a row from source of truth.
			ret.MissingTables = append(
				ret.MissingTables,
				utils.MissingTable{DBTable: truthIterator.curr()},
			)
			truthIterator.next()
		}
	}

	for !nonTruthIterator.done() {
		ret.ExtraneousTables = append(
			ret.ExtraneousTables,
			utils.ExtraneousTable{DBTable: nonTruthIterator.curr()},
		)
		nonTruthIterator.next()
	}
	return ret
}
