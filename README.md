A small tool for playing around with osm databases (those resulting
from `osm2pgsql`). Read `SETUP_DATABASE.md` to learn how to set up the
required database.

Note that `osm2pgsql` by default does not put all available tags into
the database and osmar only deals with this limited tag-set.

# Installation
```bash
git clone git@github.com:codesoap/osmar.git
cd osmar
go install
# The binary is now at ~/go/bin/osmar.
```

If you don't want to install the go compiler, you can download binaries
from the
[latest release page](https://github.com/codesoap/osmar/releases/tag/v2.1.0).

# Basic Usage
```console
$ # Find all entries within 50m of the center of Bremen, Germany:
$ osmar 53.076 8.807 50
table: planet_osm_polygon
distance_meters: 0
osm_id: -3133460
osm_link: https://www.openstreetmap.org/relation/3133460
boundary: political
name: Bremen I
ref: 54
way_area: 427011008.000000

table: planet_osm_polygon
distance_meters: 0
osm_id: -4496501
osm_link: https://www.openstreetmap.org/relation/4496501
access: green_sticker_germany
boundary: low_emission_zone
name: Umweltzone Bremen
way_area: 19706300.000000
...

$ # Use UNIX tools to compact the output:
$ osmar 53.076 8.807 50 | awk '/^$/ /^(distance|osm_link|name)/'
distance_meters: 0
osm_link: https://www.openstreetmap.org/relation/3133460
name: Bremen I

distance_meters: 0
osm_link: https://www.openstreetmap.org/relation/4496501
name: Umweltzone Bremen
...

$ # Find a bicycle shop near the center of Bremen:
$ osmar 53.076 8.807 500 shop=bicycle | awk '/^(table|osm_id):/ {next} //'
distance_meters: 244
osm_link: https://www.openstreetmap.org/node/834082330
addr:housenumber: 30-32
name: Velo-Sport
operator: Velo-Sport Ihr Radsporthaus GmbH
shop: bicycle
```

# More Examples
```bash
# Find a natural forest of at least 1km²:
osmar 53.076 8.807 3300 natural=wood 'way_area>1e+6'

# Find a bakery:
osmar 53.076 8.807 200 shop=bakery

# Find nearby public transport stations:
osmar 53.076 8.807 200 public_transport=stop_position

# Find nearby hiking routes:
osmar 53.076 8.807 500 route=hiking

# Searching for multiple values of the same tag is also possible:
osmar 53.076 8.807 3000 sport=climbing sport=swimming

# Pro tip: Use "_" to search for any value:
osmar 53.076 8.807 500 sport=_

# Learn about the population of the city and it's urban districts:
osmar 53.076 8.807 10000 population=_
```

# Custom Database Connection
A custom database connection can be used by setting
the `OSMAR_CONN` environment variable; find more info
[here](https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING):

```console
OSMAR_CONN='postgres://user:password@localhost:5432/gis' osmar 53.076 8.807 50
```

# Full Usage Info
```
osmar <lat> <long> <radius_meter> [way_area<<value>] [way_area><value>] [<tag>=<value>]...

Environment:
	OSMAR_CONN  Custom connection string for the PostgreSQL database.

General tags:
	access
	addr:housename
	addr:housenumber
	addr:interpolation
	admin_level
	aerialway
	aeroway
	amenity
	area
	barrier
	bicycle
	brand
	bridge
	boundary
	building
	construction
	covered
	culvert
	cutting
	denomination
	disused
	embankment
	foot
	generator:source
	harbour
	highway
	historic
	horse
	intermittent
	junction
	landuse
	layer
	leisure
	lock
	man_made
	military
	motorcar
	name
	natural
	office
	oneway
	operator
	place
	population
	power
	power_source
	public_transport
	railway
	ref
	religion
	route
	service
	shop
	sport
	surface
	toll
	tourism
	tower:type
	tunnel
	water
	waterway
	wetland
	width
	wood

Tags only for lines and polygons:
	tracktype

Tags only for for points:
	capital
	ele
```

The tags are explained
[here](https://wiki.openstreetmap.org/wiki/Map_Features).
