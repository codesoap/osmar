package main

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/codesoap/pbf"
)

const usage = `Usage: osmar <lat> <lon> <radius_meter> [<tag>=<value>]...
Info about tags: https://wiki.openstreetmap.org/wiki/Map_Features

Environment:
	OSMAR_PBF_FILE  The path to the PBF file.
`

var pbfFile = ""

type entity struct {
	e        *pbf.Entity
	distance int // distance in meters
}

func init() {
	if pbfFile = os.Getenv("OSMAR_PBF_FILE"); pbfFile == "" {
		fmt.Fprintln(os.Stderr, "The OSMAR_PBF_FILE environment variable must be set.")
		os.Exit(1)
	}
}

func main() {
	if len(os.Args) < 4 {
		fmt.Fprintf(os.Stderr, usage)
		os.Exit(1)
	}
	lat, err := strconv.ParseFloat(os.Args[1], 64)
	dieOnErr("Could not parse lat: %s\n", err)
	lon, err := strconv.ParseFloat(os.Args[2], 64)
	dieOnErr("Could not parse lon: %s\n", err)
	radius, err := strconv.ParseFloat(os.Args[3], 64)
	dieOnErr("Could not parse radius: %s\n", err)
	tags, err := getTags()
	dieOnErr("Could not parse tags: %s\n", err)

	res, err := getResults(lat, lon, radius, tags)
	dieOnErr("Failed to query database: %s\n", err)
	sort.Slice(res, func(i, j int) bool { return res[i].distance < res[j].distance })
	printResults(res)
}

func dieOnErr(msg string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, msg, err.Error())
		os.Exit(1)
	}
}

func getTags() (map[string][]string, error) {
	tags := make(map[string][]string)
	for _, arg := range os.Args[4:] {
		split := strings.SplitN(arg, "=", 2)
		if len(split) != 2 {
			err := fmt.Errorf("tag without value: %s", arg)
			return nil, err
		}
		if split[1] == "*" {
			tags[split[0]] = []string{}
		} else {
			tags[split[0]] = append(tags[split[0]], split[1])
		}
	}
	return tags, nil
}

func getResults(lat, lon, radius float64, tags map[string][]string) ([]entity, error) {
	// convert meters roughly to nanodegrees:
	radiusLatInt := int64(1_000_000_000 * radius / 111_000)
	radiusLonInt := int64(1_000_000_000 * radius / (6_367_000 * math.Cos(lat*math.Pi/180) * math.Pi / 180))

	latInt := int64(1_000_000_000 * lat) // convert to nanodegrees
	lonInt := int64(1_000_000_000 * lon) // convert to nanodegrees
	maxLat, minLat := latInt+radiusLatInt, latInt-radiusLatInt
	maxLon, minLon := lonInt+radiusLonInt, lonInt-radiusLonInt
	locFilter := func(lat, lon int64) bool {
		// Just do a square here; expensive (sqrt) filtering is done again
		// later, when the entities have been "pre-filtered".
		return lat >= minLat && lat <= maxLat &&
			lon >= minLon && lon <= maxLon
	}
	filter := pbf.Filter{
		Location:       locFilter,
		ExcludePartial: true,
		Tags:           tags,
	}
	entities, err := pbf.ExtractEntities(pbfFile, filter)
	if err != nil {
		return nil, err
	}
	ret := make([]entity, 0, len(entities.Nodes)+len(entities.Ways)+len(entities.Relations))
	radiusInt := int(radius)
	for _, e := range entities.Nodes {
		ent := pbf.Entity(e)
		dist := getDistance(latInt, lonInt, ent, entities)
		if dist <= radiusInt {
			// Filtering by radius again, as we just did a square filter
			// for performance earlier.
			ret = append(ret, entity{e: &ent, distance: dist})
		}
	}
	for _, e := range entities.Ways {
		ent := pbf.Entity(e)
		dist := getDistance(latInt, lonInt, ent, entities)
		if dist <= radiusInt {
			// Filtering by radius again, as we just did a square filter
			// for performance earlier.
			ret = append(ret, entity{e: &ent, distance: dist})
		}
	}
	for _, e := range entities.Relations {
		ent := pbf.Entity(e)
		dist := getDistance(latInt, lonInt, ent, entities)
		if dist <= radiusInt {
			// Filtering by radius again, as we just did a square filter
			// for performance earlier.
			ret = append(ret, entity{e: &ent, distance: dist})
		}
	}
	return ret, nil
}

// getDistance determines the distance in meters from latA, lonA to the
// closest point of e.
//
// TODO: Use ancillary entities to determine distance, once this
// feature is available in github.com/codesoap/pbf. Right now, ways and
// relations will often have an unknown distance, because their members
// didn't match the filter.
func getDistance(latA, lonA int64, e pbf.Entity, entities pbf.Entities) int {
	closest := -1
	switch t := e.(type) {
	case pbf.Node:
		latB, lonB := t.Coords()
		return calculateDistance(latA, lonA, latB, lonB)
	case pbf.Way:
		for _, nodeID := range t.Nodes() {
			if node, ok := entities.Nodes[nodeID]; ok {
				latB, lonB := node.Coords()
				dist := calculateDistance(latA, lonA, latB, lonB)
				if closest == -1 || dist < closest {
					closest = dist
				}
			}
		}
	case pbf.Relation:
		for _, nodeID := range t.Nodes() {
			if node, ok := entities.Nodes[nodeID]; ok {
				latB, lonB := node.Coords()
				dist := calculateDistance(latA, lonA, latB, lonB)
				if closest == -1 || dist < closest {
					closest = dist
				}
			}
		}
		for _, wayID := range t.Ways() {
			if way, ok := entities.Ways[wayID]; ok {
				e2 := pbf.Entity(way)
				dist := getDistance(latA, lonA, e2, entities)
				if closest == -1 || dist < closest {
					closest = dist
				}
			}
		}
		for _, relationID := range t.Relations() {
			if relation, ok := entities.Relations[relationID]; ok {
				e2 := pbf.Entity(relation)
				dist := getDistance(latA, lonA, e2, entities)
				if closest == -1 || dist < closest {
					closest = dist
				}
			}
		}
	}
	return closest
}

func calculateDistance(latA, lonA, latB, lonB int64) int {
	latAf := float64(latA) / 1_000_000_000
	latBf := float64(latB) / 1_000_000_000
	lonAf := float64(lonA) / 1_000_000_000
	lonBf := float64(lonB) / 1_000_000_000
	y := (latBf - latAf) * 111_000
	x := (lonBf - lonAf) * 6_367_000 * math.Cos(latAf*math.Pi/180) * math.Pi / 180
	return int(math.Sqrt(y*y + x*x))
}

func printResults(entities []entity) {
	for i, entityWithDist := range entities {
		entity := *entityWithDist.e
		if i > 0 {
			fmt.Println()
		}
		eType := "unknown"
		switch entity.(type) {
		case pbf.Node:
			eType = "node"
		case pbf.Way:
			eType = "way"
		case pbf.Relation:
			eType = "relation"
		}
		if entityWithDist.distance >= 0 {
			fmt.Printf("meta:distance: %dm\n", entityWithDist.distance)
		} else {
			fmt.Println("meta:distance: unknown")
		}
		fmt.Println("meta:id:", entity.ID())
		fmt.Println("meta:type:", eType)
		fmt.Printf("meta:link: https://www.openstreetmap.org/%s/%d\n", eType, entity.ID())
		tags := entity.Tags()
		keys := make([]string, 0, len(tags))
		for k := range tags {
			keys = append(keys, k)
		}
		sort.StringSlice(keys).Sort()
		for _, tag := range keys {
			fmt.Printf("%s: %s\n", tag, tags[tag])
		}
	}
}
