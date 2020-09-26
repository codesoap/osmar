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
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open database connection: %s\n", err.Error())
		os.Exit(1)
	}
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
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not parse lat: %s\n", err.Error())
		os.Exit(1)
	}
	long, err := strconv.ParseFloat(os.Args[3], 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not parse long: %s\n", err.Error())
		os.Exit(1)
	}
	radius, err := strconv.ParseFloat(os.Args[4], 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not parse radius: %s\n", err.Error())
		os.Exit(1)
	}

	tags := make(map[string][]string)
	for _, tag := range os.Args[5:] {
		split := strings.SplitN(tag, "=", 2)
		if len(split) != 2 {
			fmt.Fprintf(os.Stderr, "Could not parse tag: %s\n", tag)
			os.Exit(1)
		}
		tags[split[0]] = append(tags[split[0]], split[1])
	}

	query := fmt.Sprintf("SELECT * FROM planet_osm_%s", os.Args[1])
	poly := getBoundaryPolygon(lat, long, radius)
	query += fmt.Sprintf("\nWHERE way && ST_Transform(ST_GeomFromText('%s', 4326), 3857)", poly)
	query += getTagsFilter(tags)

	rows, err := pool.Query(query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to query database: %s\n", err.Error())
		os.Exit(1)
	}
	columnNames, err := rows.Columns()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read column names: %s\n", err.Error())
		os.Exit(1)
	}
	columns := make([]interface{}, len(columnNames))
	columnPointers := make([]interface{}, len(columnNames))
	for i := range columns {
		columnPointers[i] = &columns[i]
	}
	firstRow := true
	for rows.Next() {
		if !firstRow {
			fmt.Println("")
		}
		if err := rows.Scan(columnPointers...); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read row: %s\n", err.Error())
			os.Exit(1)
		}
		for i, columnName := range columnNames {
			if columnName == "z_order" || columnName == "way_area" || columnName == "way" {
				// Those columns are not for displaying.
			} else if columnName == "osm_id" {
				val := columnPointers[i].(*interface{})
				id := (*val).(int64)
				fmt.Printf("%s: %d\n", columnName, id)
				if id < 0 {
					id = -id // Seems to be necessary for the links.
				}
				switch os.Args[1] {
				case "point":
					fmt.Printf("osm_link: https://www.openstreetmap.org/node/%d\n", id)
				case "line":
					fmt.Printf("osm_link: https://www.openstreetmap.org/way/%d\n", id)
				case "polygon":
					fmt.Printf("osm_link: https://www.openstreetmap.org/relation/%d\n", id)
				}
			} else {
				val := columnPointers[i].(*interface{})
				valString, _ := (*val).(string) // Second return value is used to accept nil.
				fmt.Printf("%s: %s\n", columnName, valString)
			}
		}
		firstRow = false
	}
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
	if len(tags) > 0 {
	}
	for tag, values := range tags {
		filter += "\nAND ("
		for i, value := range values {
			filter += fmt.Sprintf(" %s LIKE '%%%s%%'", tag, value)
			if len(values) > i+1 {
				filter += " OR"
			}
		}
		filter += ")"
	}
	return
}
