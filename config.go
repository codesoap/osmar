package main

const dataSourceName = "host=localhost port=5432 database=gis"

// The more corners the polygon has, the closer the boundary of the
// search will resemble a circle, but also the slower the query will be.
const boundaryPolygonCorners = 8
