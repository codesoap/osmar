package main

import (
	"database/sql"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	_ "github.com/jackc/pgx/v4/stdlib"
)

const usage = `Usage:
osmf line <lat> <long> <radius_meter> [<tag>=<value>]...
osmf point <lat> <long> <radius_meter> [<tag>=<value>]...
osmf polygon <lat> <long> <radius_meter> [<tag>=<value>]...
`

var pool *sql.DB

func init() {
	var err error
	pool, err = sql.Open("pgx", "host=localhost port=5432 database=gis")
	dieOnErr("Failed to open database connection: %s\n", err)
}

func main() {
	defer pool.Close()

	if len(os.Args) < 5 {
		fmt.Fprintf(os.Stderr, usage)
		os.Exit(1)
	}
	if os.Args[1] != "line" && os.Args[1] != "point" && os.Args[1] != "polygon" {
		fmt.Fprintf(os.Stderr, "Invalid subcommand: %s\n", os.Args[1])
		os.Exit(1)
	}

	lat, err := strconv.ParseFloat(os.Args[2], 64)
	dieOnErr("Could not parse lat: %s\n", err)
	long, err := strconv.ParseFloat(os.Args[3], 64)
	dieOnErr("Could not parse long: %s\n", err)
	radius, err := strconv.ParseFloat(os.Args[4], 64)
	dieOnErr("Could not parse radius: %s\n", err)
	tags, err := getTags()
	dieOnErr("Could not parse tags: %s\n", err)

	rows, err := queryDB(lat, long, radius, tags)
	dieOnErr("Failed to query database: %s\n", err)
	dieOnErr("Failed to print results: %s\n", printResults(rows))
}

func dieOnErr(msg string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, msg, err.Error())
		os.Exit(1)
	}
}

func getTags() (map[string][]string, error) {
	tags := make(map[string][]string)
	for _, tag := range os.Args[5:] {
		split := strings.SplitN(tag, "=", 2)
		if len(split) != 2 {
			return tags, fmt.Errorf("tag without value: %s", tag)
		}
		tags[split[0]] = append(tags[split[0]], split[1])
	}
	return tags, nil
}

func queryDB(lat, long, radius float64, tags map[string][]string) (*sql.Rows, error) {
	refPoint := fmt.Sprintf("ST_SetSRID(ST_Point(%f, %f), 4326)::geography", long, lat)
	distance := fmt.Sprintf("ST_Distance(ST_Transform(way, 4326)::geography, %s) AS distance", refPoint)
	query := fmt.Sprintf("SELECT %s, * FROM planet_osm_%s\n", distance, os.Args[1])
	poly := getBoundaryPolygon(lat, long, radius)
	query += fmt.Sprintf("WHERE way && ST_Transform(ST_GeomFromText('%s', 4326), 3857)", poly)
	query += getTagsFilter(tags)
	query += "\nORDER BY distance"
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

func printResults(rows *sql.Rows) error {
	colNames, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to read col names: %s\n", err.Error())
	}
	cols := make([]interface{}, len(colNames))
	colPtrs := make([]interface{}, len(colNames))
	for i := range cols {
		colPtrs[i] = &cols[i]
	}
	firstRow := true
	for rows.Next() {
		if !firstRow {
			fmt.Println("")
		}
		if err = printResult(colNames, colPtrs, rows); err != nil {
			return err
		}
		firstRow = false
	}
	return nil
}

func printResult(colNames []string, colPtrs []interface{}, rows *sql.Rows) error {
	if err := rows.Scan(colPtrs...); err != nil {
		return fmt.Errorf("failed to read row: %s\n", err.Error())
	}
	for i, colName := range colNames {
		if colName == "z_order" || colName == "way_area" || colName == "way" {
			// Those columns are not for displaying.
		} else if colName == "distance" {
			val := colPtrs[i].(*interface{})
			fmt.Printf("distance_meters: %.0f\n", (*val).(float64))
		} else if colName == "osm_id" {
			val := colPtrs[i].(*interface{})
			id := (*val).(int64)
			fmt.Printf("%s: %d\n", colName, id)
			switch os.Args[1] {
			case "point":
				fmt.Printf("osm_link: https://www.openstreetmap.org/node/%d\n", id)
			case "line":
				fmt.Printf("osm_link: https://www.openstreetmap.org/way/%d\n", id)
			case "polygon":
				// Relations have negative IDs.
				// See https://help.openstreetmap.org/questions/2259
				if id < 0 {
					fmt.Printf("osm_link: https://www.openstreetmap.org/relation/%d\n", -id)
				} else {
					fmt.Printf("osm_link: https://www.openstreetmap.org/way/%d\n", id)
				}
			}
		} else {
			val := colPtrs[i].(*interface{})
			valString, _ := (*val).(string) // Second return value is used to accept nil.
			fmt.Printf("%s: %s\n", colName, valString)
		}
	}
	return nil
}
