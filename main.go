package main

import (
	"database/sql"
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	_ "github.com/jackc/pgx/v4/stdlib"
)

const usage = `Usage: osmf <lat> <long> <radius_meter> [way_area<<value>] [way_area><value>] [<tag>=<value>]...
Info about tags: https://wiki.openstreetmap.org/wiki/Map_Features

Environment:
	OSMF_CONN  Custom connection string for the PostgreSQL database.
`

var pool *sql.DB
var nonColumNameRe = regexp.MustCompile(`[^a-zA-Z_:]+`)

type osmTableType int

const (
	osmPointTable = iota
	osmLineTable
	osmPolygonTable
)

// row represents one row of a query on the point, line or polygon
// table.
type row struct {
	tableType osmTableType
	distance  float64        // Distance to the given coordinates in meter.
	values    []*interface{} // Values for all columns, including distance.
}

// results represents the SQL query results on the point, line and
// polygon tables.
type results struct {
	pointColNames   []string
	lineColNames    []string
	polygonColNames []string
	rows            []row
}

type byDistance []row

func (a byDistance) Len() int           { return len(a) }
func (a byDistance) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byDistance) Less(i, j int) bool { return a[i].distance < a[j].distance }

func init() {
	dataSourceName := os.Getenv("OSMF_CONN")
	if dataSourceName == "" {
		dataSourceName = defaultDataSourceName
	}
	var err error
	pool, err = sql.Open("pgx", dataSourceName)
	dieOnErr("Failed to open database connection: %s\n", err)
}

func main() {
	defer pool.Close()

	if len(os.Args) < 4 {
		fmt.Fprintf(os.Stderr, usage)
		os.Exit(1)
	}
	lat, err := strconv.ParseFloat(os.Args[1], 64)
	dieOnErr("Could not parse lat: %s\n", err)
	long, err := strconv.ParseFloat(os.Args[2], 64)
	dieOnErr("Could not parse long: %s\n", err)
	radius, err := strconv.ParseFloat(os.Args[3], 64)
	dieOnErr("Could not parse radius: %s\n", err)
	tags, minWayArea, maxWayArea, err := getFilters()
	dieOnErr("Could not parse filters: %s\n", err)

	res, err := getResults(lat, long, radius, tags, minWayArea, maxWayArea)
	dieOnErr("Failed to query database: %s\n", err)
	sort.Sort(byDistance(res.rows))
	printResults(res)
}

func dieOnErr(msg string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, msg, err.Error())
		os.Exit(1)
	}
}

func getFilters() (tags map[string][]string, minWayArea, maxWayArea *float64, err error) {
	tags = make(map[string][]string)
	for _, arg := range os.Args[4:] {
		if strings.HasPrefix(arg, "way_area>") {
			var minWayAreaTmp float64
			minWayAreaTmp, err = strconv.ParseFloat(arg[9:], 64)
			if err != nil {
				return
			}
			minWayArea = &minWayAreaTmp
		} else if strings.HasPrefix(arg, "way_area<") {
			var maxWayAreaTmp float64
			maxWayAreaTmp, err = strconv.ParseFloat(arg[9:], 64)
			if err != nil {
				return
			}
			maxWayArea = &maxWayAreaTmp
		} else {
			split := strings.SplitN(arg, "=", 2)
			if len(split) != 2 {
				err = fmt.Errorf("tag without value: %s", arg)
				return
			}
			tags[split[0]] = append(tags[split[0]], split[1])
		}
	}
	return
}

func getResults(lat, long, radius float64, tags map[string][]string, minWayArea, maxWayArea *float64) (results, error) {
	var res results
	var err error

	// Certain tags don't appear in all tables:
	_, skipPointsTable := tags["tracktype"]
	_, skipLineAndPolygonTables := tags["capital"]
	if _, ok := tags["ele"]; ok {
		skipLineAndPolygonTables = true
	}

	if minWayArea == nil && maxWayArea == nil && !skipPointsTable {
		// Search for results in the point table:
		points, err := queryDB(lat, long, radius, tags, nil, nil, "point")
		if err != nil {
			return res, err
		}
		if res.pointColNames, err = points.Columns(); err != nil {
			return res, fmt.Errorf("failed to read col names: %s\n", err.Error())
		}
		err = fillRowsOfType(points, &res.rows, len(res.pointColNames), osmPointTable)
		if err != nil {
			return res, err
		}
	}

	if !skipLineAndPolygonTables {
		// Search for results in the line table:
		lines, err := queryDB(lat, long, radius, tags, minWayArea, maxWayArea, "line")
		if err != nil {
			return res, err
		}
		if res.lineColNames, err = lines.Columns(); err != nil {
			return res, fmt.Errorf("failed to read col names: %s\n", err.Error())
		}
		err = fillRowsOfType(lines, &res.rows, len(res.lineColNames), osmLineTable)
		if err != nil {
			return res, err
		}

		// Search for results in the polygon table:
		polygons, err := queryDB(lat, long, radius, tags, minWayArea, maxWayArea, "polygon")
		if err != nil {
			return res, err
		}
		if res.polygonColNames, err = polygons.Columns(); err != nil {
			return res, fmt.Errorf("failed to read col names: %s\n", err.Error())
		}
		err = fillRowsOfType(polygons, &res.rows, len(res.polygonColNames), osmPolygonTable)
	}
	return res, err
}

func fillRowsOfType(dbRows *sql.Rows, resRows *[]row, colCnt int, resType osmTableType) error {
	for i := 0; dbRows.Next(); i += 1 {
		cols := make([]interface{}, colCnt)
		colPtrs := make([]interface{}, colCnt)
		for i := range cols {
			colPtrs[i] = &cols[i]
		}
		if err := dbRows.Scan(colPtrs...); err != nil {
			return fmt.Errorf("failed to read row: %s", err.Error())
		}
		var row row
		row.tableType = resType
		row.values = make([]*interface{}, colCnt)
		for i := range colPtrs {
			if i == 0 {
				// First column is always the distance.
				val := colPtrs[i].(*interface{})
				valFloat, ok := (*val).(float64)
				if !ok {
					return fmt.Errorf("failed to read distance")
				}
				row.distance = valFloat
			}
			row.values[i] = colPtrs[i].(*interface{})
		}
		*resRows = append(*resRows, row)
	}
	return nil
}

func queryDB(lat, long, radius float64, tags map[string][]string, minWayArea, maxWayArea *float64, table string) (*sql.Rows, error) {
	refPoint := fmt.Sprintf("ST_SetSRID(ST_Point(%f, %f), 4326)::geography", long, lat)
	distance := fmt.Sprintf("ST_Distance(ST_Transform(way, 4326)::geography, %s) AS distance", refPoint)
	query := fmt.Sprintf("SELECT %s, * FROM %s_%s\n", distance, tablePrefix, table)
	poly := getBoundaryPolygon(lat, long, radius)
	query += fmt.Sprintf("WHERE way && ST_Transform(ST_GeomFromText('%s', 4326), 3857)", poly)
	tagsFilter, params := getTagsFilter(tags)
	query += tagsFilter
	query += getWayAreaFilter(minWayArea, maxWayArea)
	return pool.Query(query, params...)
}

func getBoundaryPolygon(lat, long, radius float64) string {
	radiusDeg := radius / 111000 // One degree is ca. 111km
	poly := "POLYGON(("
	for i := 0; i <= boundaryPolygonCorners; i += 1 {
		cornerLat := lat + radiusDeg*math.Sin(2*math.Pi*float64(i)/float64(boundaryPolygonCorners))
		cornerLong := long + radiusDeg*math.Cos(2*math.Pi*float64(i)/float64(boundaryPolygonCorners))
		poly += fmt.Sprintf(" %f %f", cornerLong, cornerLat)
		if boundaryPolygonCorners > i {
			poly += ","
		}
	}
	poly += "))"
	return poly
}

func getTagsFilter(tags map[string][]string) (filter string, params []interface{}) {
	paramIndex := 1
	for tag, values := range tags {
		filter += "\nAND ("
		for i, value := range values {
			// FIXME: Update when https://github.com/golang/go/issues/18478 is fixed.
			tag = nonColumNameRe.ReplaceAllString(tag, "")

			filter += fmt.Sprintf(` LOWER("%s") LIKE LOWER('%%' || $%d || '%%')`, tag, paramIndex)
			paramIndex++
			params = append(params, value)
			if len(values) > i+1 {
				filter += " OR"
			}
		}
		filter += ")"
	}
	return
}

func getWayAreaFilter(minWayArea, maxWayArea *float64) (filter string) {
	if minWayArea != nil {
		filter += fmt.Sprintf("\nAND way_area > %f", *minWayArea)
	}
	if maxWayArea != nil {
		filter += fmt.Sprintf("\nAND way_area < %f", *maxWayArea)
	}
	return
}

func printResults(res results) {
	firstRow := true
	for _, resRow := range res.rows {
		if !firstRow {
			fmt.Println("")
		}
		switch resRow.tableType {
		case osmPointTable:
			printResult(res.pointColNames, resRow)
		case osmLineTable:
			printResult(res.lineColNames, resRow)
		case osmPolygonTable:
			printResult(res.polygonColNames, resRow)
		}
		firstRow = false
	}
}

func printResult(colNames []string, resRow row) {
	switch resRow.tableType {
	case osmPointTable:
		fmt.Printf("table: %s_point\n", tablePrefix)
	case osmLineTable:
		fmt.Printf("table: %s_line\n", tablePrefix)
	case osmPolygonTable:
		fmt.Printf("table: %s_polygon\n", tablePrefix)
	}
	for i, colName := range colNames {
		if colName == "z_order" || colName == "way" {
			// Those columns are not for displaying.
		} else if colName == "way_area" {
			if valFloat, ok := (*resRow.values[i]).(float64); ok {
				fmt.Printf("%s: %f\n", colName, valFloat)
			}
		} else if colName == "distance" {
			fmt.Printf("distance_meters: %.0f\n", (*resRow.values[i]).(float64))
		} else if colName == "osm_id" {
			id := (*resRow.values[i]).(int64)
			fmt.Printf("%s: %d\n", colName, id)
			if resRow.tableType == osmPointTable {
				fmt.Printf("osm_link: https://www.openstreetmap.org/node/%d\n", id)
			} else {
				// Relations have negative IDs.
				// See https://help.openstreetmap.org/questions/2259
				if id < 0 {
					fmt.Printf("osm_link: https://www.openstreetmap.org/relation/%d\n", -id)
				} else {
					fmt.Printf("osm_link: https://www.openstreetmap.org/way/%d\n", id)
				}
			}
		} else {
			valString, _ := (*resRow.values[i]).(string) // Second return value is used to accept nil.
			if len(valString) > 0 {
				fmt.Printf("%s: %s\n", colName, valString)
			}
		}
	}
}
