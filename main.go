package main

import (
	"database/sql"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	_ "github.com/jackc/pgx/v4/stdlib"
)

var verbose = flag.Bool("v", false, "verbose mode; also output empty values")

var pool *sql.DB

type osmTableType int

const (
	osmPointTable = iota
	osmLineTable
	osmPolygonTable
)

// row represents one row of a query on the planet_osm_point,
// planet_osm_line or planet_osm_polygon table.
type row struct {
	tableType osmTableType
	distance  float64        // Distance to the given coordinates in meter.
	values    []*interface{} // Values for all columns, including distance.
}

// results represents the SQL query results on the planet_osm_point,
// planet_osm_line and planet_osm_polygon tables.
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
	var err error
	pool, err = sql.Open("pgx", "host=localhost port=5432 database=gis")
	dieOnErr("Failed to open database connection: %s\n", err)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage:
osmf [-v] <lat> <long> <radius_meter> [way_area<<value>] [way_area><value>] [<tag>=<value>]...
Options:
	-v verbose mode; also output empty values
Info about tags: https://wiki.openstreetmap.org/wiki/Map_Features
`)
	}
	flag.Parse()
}

func main() {
	defer pool.Close()

	if len(flag.Args()) < 3 {
		flag.Usage()
		os.Exit(1)
	}
	lat, err := strconv.ParseFloat(flag.Arg(0), 64)
	dieOnErr("Could not parse lat: %s\n", err)
	long, err := strconv.ParseFloat(flag.Arg(1), 64)
	dieOnErr("Could not parse long: %s\n", err)
	radius, err := strconv.ParseFloat(flag.Arg(2), 64)
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
	for _, arg := range flag.Args()[3:] {
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
		// Search for results in the planet_osm_point table:
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
		// Search for results in the planet_osm_line table:
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

		// Search for results in the planet_osm_polygon table:
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
	query := fmt.Sprintf("SELECT %s, * FROM planet_osm_%s\n", distance, table)
	poly := getBoundaryPolygon(lat, long, radius)
	query += fmt.Sprintf("WHERE way && ST_Transform(ST_GeomFromText('%s', 4326), 3857)", poly)
	query += getTagsFilter(tags)
	query += getWayAreaFilter(minWayArea, maxWayArea)
	return pool.Query(query)
}

func getBoundaryPolygon(lat, long, radius float64) string {
	radiusDeg := radius / 111000 // One degree is ca. 111km
	poly := "POLYGON(("
	corners := 8
	for i := 0; i <= corners; i += 1 {
		cornerLat := lat + radiusDeg*math.Sin(2*math.Pi*float64(i)/float64(corners))
		cornerLong := long + radiusDeg*math.Cos(2*math.Pi*float64(i)/float64(corners))
		poly += fmt.Sprintf(" %f %f", cornerLong, cornerLat)
		if corners > i {
			poly += ","
		}
	}
	poly += "))"
	return poly
}

func getTagsFilter(tags map[string][]string) (filter string) {
	for tag, values := range tags {
		filter += "\nAND ("
		for i, value := range values {
			// Poor mans SQL escaping for simplicity:
			tag = strings.ReplaceAll(tag, `"`, "")
			value = strings.ReplaceAll(value, `'`, "")

			filter += fmt.Sprintf(` "%s" LIKE '%%%s%%'`, tag, value)
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
		fmt.Printf("table: planet_osm_point\n")
	case osmLineTable:
		fmt.Printf("table: planet_osm_line\n")
	case osmPolygonTable:
		fmt.Printf("table: planet_osm_polygon\n")
	}
	for i, colName := range colNames {
		if colName == "z_order" || colName == "way" {
			// Those columns are not for displaying.
		} else if colName == "way_area" {
			valFloat, ok := (*resRow.values[i]).(float64)
			if ok {
				fmt.Printf("%s: %f\n", colName, valFloat)
			} else if *verbose {
				fmt.Printf("%s:\n", colName)
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
			if len(valString) > 0 || *verbose {
				fmt.Printf("%s: %s\n", colName, valString)
			}
		}
	}
}
