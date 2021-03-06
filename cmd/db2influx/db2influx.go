package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	lib "gha2db"
)

// Returns multi row and multi column series names array (differen for different rows)
// Each row must be in format: 'prefix;name;series1,series2,..,seriesN' serVal1 serVal2 ... serValN
func multiRowMultiColumn(col, period string) (result []string) {
	ary := strings.Split(col, ";")
	r := strings.NewReplacer("-", "_", "/", "_", ".", "_", " ", "_")
	name := r.Replace(strings.ToLower(strings.TrimSpace(lib.StripUnicode(ary[1]))))
	if name == "" {
		lib.Printf("multiRowMultiColumn: WARNING: name '%v' (%+v) maps to empty string, skipping\n", ary[1], ary)
		return
	}
	splitted := strings.Split(ary[2], ",")
	if ary[0] != "" {
		pref := ary[0]
		for _, series := range splitted {
			result = append(result, fmt.Sprintf("%s_%s_%s_%s", pref, name, series, period))
		}
	} else {
		for _, series := range splitted {
			result = append(result, fmt.Sprintf("%s_%s_%s", name, series, period))
		}
	}
	return
}

// Return default series names from multi column single row result
// It takes name "a,b,c,d,...,z" and period for example "q"
// and returns array [a_q, b_q, c_q, .., z_q]
func singleRowMultiColumn(col, period string) (result []string) {
	splitted := strings.Split(col, ",")
	for _, str := range splitted {
		result = append(result, str+"_"+period)
	}
	return
}

// Return default series names from multi row result single column
// Each row is "prefix,name", value (prefix is hardcoded in metric, so it is assumed safe)
// and returns array [a_q, b_q, c_q, .., z_q]
func multiRowSingleColumn(col, period string) (result []string) {
	ary := strings.Split(col, ",")
	r := strings.NewReplacer("-", "_", "/", "_", ".", "_", " ", "_")
	name := r.Replace(strings.ToLower(strings.TrimSpace(lib.StripUnicode(ary[1]))))
	if name == "" {
		lib.Printf("multiRowSingleColumn: WARNING: name '%v' (%+v) maps to empty string, skipping\n", ary[1], ary)
		return
	}
	if ary[0] != "" {
		name = ary[0] + "_" + name
	}
	return []string{fmt.Sprintf("%s_%s", name, period)}
}

// Generate name for given series row and period
func nameForMetricsRow(metric, name, period string) []string {
	switch metric {
	case "single_row_multi_column":
		return singleRowMultiColumn(name, period)
	case "multi_row_single_column":
		return multiRowSingleColumn(name, period)
	case "multi_row_multi_column":
		return multiRowMultiColumn(name, period)
	default:
		lib.Printf("Error\nUnknown metric '%v'\n", metric)
		fmt.Fprintf(os.Stdout, "Error\nUnknown metric '%v'\n", metric)
		os.Exit(1)
	}
	return []string{""}
}

// Round float64 to int
func roundF2I(val float64) int {
	if val < 0.0 {
		return int(val - 0.5)
	}
	return int(val + 0.5)
}

func workerThread(ch chan bool, ctx *lib.Ctx, seriesNameOrFunc, sqlQuery, period string, from, to time.Time) {
	// Connect to Postgres DB
	sqlc := lib.PgConn(ctx)
	defer sqlc.Close()

	// Connect to InfluxDB
	ic := lib.IDBConn(ctx)
	defer ic.Close()

	// Get BatchPoints
	bp := lib.IDBBatchPoints(ctx, &ic)

	// Prepare SQL query
	sFrom := lib.ToYMDHMSDate(from)
	sTo := lib.ToYMDHMSDate(to)
	sqlQuery = strings.Replace(sqlQuery, "{{from}}", sFrom, -1)
	sqlQuery = strings.Replace(sqlQuery, "{{to}}", sTo, -1)

	// Execute SQL query
	rows := lib.QuerySQLWithErr(sqlc, ctx, sqlQuery)
	defer rows.Close()

	// Get Number of columns
	// We support either query returnign single row with single numeric value
	// Or multiple rows, each containing string (series name) and its numeric value(s)
	columns, err := rows.Columns()
	lib.FatalOnError(err)
	nColumns := len(columns)

	// Metric Results, assume they're floats
	var (
		pValue *float64
		value  float64
		name   string
	)
	// Single row & single column result
	if nColumns == 1 {
		rowCount := 0
		for rows.Next() {
			lib.FatalOnError(rows.Scan(&pValue))
			rowCount++
		}
		lib.FatalOnError(rows.Err())
		if rowCount != 1 {
			lib.Printf(
				"Error:\nQuery should return either single value or "+
					"multiple rows, each containing string and numbers\n"+
					"Got %d rows, each containing single number\nQuery:%s\n",
				rowCount, sqlQuery,
			)
		}
		// Handle nulls
		if pValue != nil {
			value = *pValue
		}
		// In this simplest case 1 row, 1 column - series name is taken directly from YAML (metrics.yaml)
		// It usually uses `add_period_to_name: true` to have _perio suffix, period{=h,d,w,m,q,y}
		name = seriesNameOrFunc
		if ctx.Debug > 0 {
			lib.Printf("%v - %v -> %v, %v\n", from, to, name, value)
		}
		// Add batch point
		fields := map[string]interface{}{"value": value}
		pt := lib.IDBNewPointWithErr(name, nil, fields, from)
		bp.AddPoint(pt)
	} else if nColumns >= 2 {
		// Multiple rows, each with (series name, value(s))
		// Number of columns
		columns, err := rows.Columns()
		lib.FatalOnError(err)
		nColumns := len(columns)
		// Alocate nColumns numeric values (first is series name)
		pValues := make([]interface{}, nColumns)
		for i := range columns {
			pValues[i] = new(sql.RawBytes)
		}
		for rows.Next() {
			// Get row values
			lib.FatalOnError(rows.Scan(pValues...))
			// Get first column name, and using it all series names
			// First column should contain nColumns - 1 names separated by ","
			name := string(*pValues[0].(*sql.RawBytes))
			names := nameForMetricsRow(seriesNameOrFunc, name, period)
			if len(names) > 0 {
				// Iterate values
				pFloats := pValues[1:]
				for idx, pVal := range pFloats {
					if pVal != nil {
						value, _ = strconv.ParseFloat(string(*pVal.(*sql.RawBytes)), 64)
					} else {
						value = 0.0
					}
					name = names[idx]
					if ctx.Debug > 0 {
						lib.Printf("%v - %v -> %v: %v, %v\n", from, to, idx, name, value)
					}
					// Add batch point
					fields := map[string]interface{}{"value": value}
					pt := lib.IDBNewPointWithErr(name, nil, fields, from)
					bp.AddPoint(pt)
				}
			}
		}
		lib.FatalOnError(rows.Err())
	}
	// Write the batch
	if !ctx.SkipIDB {
		err = ic.Write(bp)
		lib.FatalOnError(err)
	} else if ctx.Debug > 0 {
		lib.Printf("Skipping series write\n")
	}

	// Synchronize go routine
	if ch != nil {
		ch <- true
	}
}

func db2influxHistogram(ctx *lib.Ctx, seriesNameOrFunc, sqlQuery, interval string) {
	// Connect to Postgres DB
	sqlc := lib.PgConn(ctx)
	defer sqlc.Close()

	// Connect to InfluxDB
	ic := lib.IDBConn(ctx)
	defer ic.Close()

	// Get BatchPoints
	bp := lib.IDBBatchPoints(ctx, &ic)

	// Prepare SQL query
	dbInterval := "1 " + interval
	if interval == "quarter" {
		dbInterval = "3 months"
	}
	sqlQuery = strings.Replace(sqlQuery, "{{period}}", dbInterval, -1)

	// Execute SQL query
	rows := lib.QuerySQLWithErr(sqlc, ctx, sqlQuery)
	defer rows.Close()

	// Get number of columns, for histograms there should be exactly 2 columns
	columns, err := rows.Columns()
	lib.FatalOnError(err)
	nColumns := len(columns)

	// Expect 2 columns: string column with name and float column with value
	var (
		value float64
		name  string
	)
	if nColumns == 2 {
		// Drop existing data
		lib.SafeQueryIDB(ic, ctx, "drop measurement "+seriesNameOrFunc)

		// Add new data
		tm := time.Now()
		rowCount := 0
		for rows.Next() {
			lib.FatalOnError(rows.Scan(&name, &value))
			if ctx.Debug > 0 {
				lib.Printf("hist %v, %v -> %v, %v\n", seriesNameOrFunc, interval, name, value)
			}
			// Add batch point
			fields := map[string]interface{}{"name": name, "value": value}
			pt := lib.IDBNewPointWithErr(seriesNameOrFunc, nil, fields, tm)
			bp.AddPoint(pt)
			rowCount++
			tm = tm.Add(-time.Hour)
		}
		if ctx.Debug > 0 {
			lib.Printf("hist %v, %v: %v rows\n", seriesNameOrFunc, interval, rowCount)
		}
		lib.FatalOnError(rows.Err())
	} else {
		lib.FatalOnError(fmt.Errorf("histogram metric should return 2 clumns, returned %v", nColumns))
	}
	// Write the batch
	if !ctx.SkipIDB {
		err = ic.Write(bp)
		lib.FatalOnError(err)
	} else if ctx.Debug > 0 {
		lib.Printf("Skipping series write\n")
	}
}

func db2influx(seriesNameOrFunc, sqlFile, from, to, intervalAbbr string, hist bool) {
	// Environment context parse
	var ctx lib.Ctx
	ctx.Init()

	// Read SQL file.
	bytes, err := ioutil.ReadFile(sqlFile)
	lib.FatalOnError(err)
	sqlQuery := string(bytes)

	// Process interval
	interval, intervalStart, nextIntervalStart := lib.GetIntervalFunctions(intervalAbbr)

	if hist {
		db2influxHistogram(&ctx, seriesNameOrFunc, sqlQuery, interval)
		return
	}

	// Parse input dates
	dFrom := lib.TimeParseAny(from)
	dTo := lib.TimeParseAny(to)

	// Round dates to the given interval
	dFrom = intervalStart(dFrom)
	dTo = nextIntervalStart(dTo)

	// Get number of CPUs available
	thrN := lib.GetThreadsNum(&ctx)

	// Run
	lib.Printf("db2influx.go: Running (on %d CPUs): %v - %v with interval %s\n", thrN, dFrom, dTo, interval)
	dt := dFrom
	if thrN > 1 {
		chanPool := []chan bool{}
		for dt.Before(dTo) {
			ch := make(chan bool)
			chanPool = append(chanPool, ch)
			nDt := nextIntervalStart(dt)
			go workerThread(ch, &ctx, seriesNameOrFunc, sqlQuery, intervalAbbr, dt, nDt)
			dt = nDt
			if len(chanPool) == thrN {
				ch = chanPool[0]
				<-ch
				chanPool = chanPool[1:]
			}
		}
		lib.Printf("Final threads join\n")
		for _, ch := range chanPool {
			<-ch
		}
	} else {
		lib.Printf("Using single threaded version\n")
		for dt.Before(dTo) {
			nDt := nextIntervalStart(dt)
			workerThread(nil, &ctx, seriesNameOrFunc, sqlQuery, intervalAbbr, dt, nDt)
			dt = nDt
		}
	}
	// Finished
	lib.Printf("All done.\n")
}

func main() {
	dtStart := time.Now()
	if len(os.Args) < 6 {
		lib.Printf(
			"Required series name, SQL file name, from, to, period " +
				"[series_name_or_func some.sql '2015-08-03' '2017-08-21' h|d|w|m|q|y\n",
		)
		lib.Printf(
			"Series name (series_name_or_func) will become exact series name if " +
				"query return just single numeric value\n",
		)
		lib.Printf("For queries returning multiple rows 'series_name_or_func' will be used as function that\n")
		lib.Printf("receives data row and period and returns name and value(s) for it\n")
		os.Exit(1)
	}
	hist := false
	if len(os.Args) > 6 && len(os.Args[6]) > 0 && strings.ToLower(os.Args[6][0:1]) == "h" {
		hist = true
	}
	db2influx(os.Args[1], os.Args[2], os.Args[3], os.Args[4], os.Args[5], hist)
	dtEnd := time.Now()
	lib.Printf("Time: %v\n", dtEnd.Sub(dtStart))
}
