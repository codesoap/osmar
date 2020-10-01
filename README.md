A small tool for playing around with osm databases (those resulting
from `osm2pgsql`). Read `SETUP_DATABASE.md` to learn how to set up the
required database.

Note that `osm2pgsql` by default does not put all available tags into
the database and osmf only deals with this limited tag-set.

# Installation
```bash
git clone git@github.com:codesoap/osmf.git
cd osmf
go install
# The binary is now at ~/go/bin/osmf.
```

If you don't want to install the go compiler, you can download binaries
from the
[latest release page](https://github.com/codesoap/osmf/releases/tag/v1.1.0).

# Basic Usage
```console
$ # Find all points within 50m of the center of Bremen, Germany:
$ osmf point 53.076 8.807 50
distance_meters: 10
osm_id: 2523704361
osm_link: https://www.openstreetmap.org/node/2523704361
access:
addr:housename:
...

$ # Use UNIX tools to compact the output:
$ osmf point 53.076 8.807 50 | awk '/^$/ / ./'
distance_meters: 10
osm_id: 2523704361
osm_link: https://www.openstreetmap.org/node/2523704361
barrier: bollard

distance_meters: 11
osm_id: 699745130
osm_link: https://www.openstreetmap.org/node/699745130
addr:housenumber: 1
amenity: restaurant
name: Beck's am Markt
...

$ # Find a bicycle shop near the center of Bremen:
$ osmf point 53.076 8.807 500 shop=bicycle | awk '/^$/ / ./'
distance_meters: 244
osm_id: 834082330
osm_link: https://www.openstreetmap.org/node/834082330
addr:housenumber: 30-32
name: Velo-Sport
operator: Velo-Sport Ihr Radsporthaus GmbH
shop: bicycle
```

# More Examples
```bash
# Find a natural forest of at least 1km²:
osmf polygon 53.076 8.807 3300 natural=wood 'way_area>1e+6' | awk '/^$/ / ./'

# Find a bakery:
osmf point 53.076 8.807 300 shop=bakery | awk '/^$/ / ./'

# Find nearby public transport stations:
osmf point 53.076 8.807 200 public_transport=stop_position | awk '/^$/ / ./'

# Find nearby hiking routes:
osmf line 53.076 8.807 1000 route=hiking | awk '/^$/ / ./'

# Searching for multiple values of the same tag is also possible:
osmf point 53.076 8.807 15000 sport=climbing sport=swimming | awk '/^$/ / ./'

# Pro tip: Use "_" to search for any value:
osmf point 53.076 8.807 500 sport=_ | awk '/^$/ / ./'

# Learn about the population of the city and it's urban districts:
osmf polygon 53.076 8.807 10000 population=_ | awk '/^$/ /^name/ /^population/ /^osm_link/'
```

# Full Usage Info
```
osmf point <lat> <long> <radius_meter> [<tag>=<value>]...
osmf line <lat> <long> <radius_meter> [way_area<<value>] [way_area><value>] [<tag>=<value>]...
osmf polygon <lat> <long> <radius_meter> [way_area<<value>] [way_area><value>] [<tag>=<value>]...

General tags:
- access
- addr:housename
- addr:housenumber
- addr:interpolation
- admin_level
- aerialway
- aeroway
- amenity
- area
- barrier
- bicycle
- brand
- bridge
- boundary
- building
- construction
- covered
- culvert
- cutting
- denomination
- disused
- embankment
- foot
- generator:source
- harbour
- highway
- historic
- horse
- intermittent
- junction
- landuse
- layer
- leisure
- lock
- man_made
- military
- motorcar
- name
- natural
- office
- oneway
- operator
- place
- population
- power
- power_source
- public_transport
- railway
- ref
- religion
- route
- service
- shop
- sport
- surface
- toll
- tourism
- tower:type
- tunnel
- water
- waterway
- wetland
- width
- wood

Tags only for lines and polygons:
- tracktype

Tags only for for points:
- capital
- ele
```

The tags are explained
[here](https://wiki.openstreetmap.org/wiki/Map_Features).
