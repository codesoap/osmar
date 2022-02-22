package main

const defaultDataSourceName = "host=localhost port=5432 database=gis"
const tablePrefix = "planet_osm"

// The more corners the polygon has, the closer the boundary of the
// search will resemble a circle, but also the slower the query will be.
const boundaryPolygonCorners = 8
