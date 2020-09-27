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

# Examples
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
$ osmf point 53.076 8.807 50 | awk '/^$/ /[^ ]*: ./'
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
$ osmf point 53.076 8.807 500 shop=bicycle | awk '/^$/ /[^ ]*: ./'
distance_meters: 244
osm_id: 834082330
osm_link: https://www.openstreetmap.org/node/834082330
addr:housenumber: 30-32
name: Velo-Sport
operator: Velo-Sport Ihr Radsporthaus GmbH
shop: bicycle

$ # Searching for multiple values of the same tag is also possible:
$ osmf point 53.076 8.807 15000 sport=climbing sport=swimming | awk '/^$/ /[^ ]*: ./'
distance_meters: 3169
osm_id: 486137250
osm_link: https://www.openstreetmap.org/node/486137250
name: Klettergarten Bremen
sport: climbing
...
distance_meters: 7048
osm_id: 3063163381
osm_link: https://www.openstreetmap.org/node/3063163381
addr:housenumber: 160
leisure: club
name: Bremischer Schwimmverein e.V.
sport: swimming;tennis

$ # Pro tip: Use "_" to search for any value:
$ osmf point 53.076 8.807 500 sport=_ | awk '/^$/ /[^ ]*: ./'
distance_meters: 342
osm_id: 4715819785
osm_link: https://www.openstreetmap.org/node/4715819785
name: Absolute Run
shop: sports
sport: running
```

# Usage
```
osmf line <lat> <long> <radius_meter> [<tag>=<value>]...
osmf point <lat> <long> <radius_meter> [<tag>=<value>]...
osmf polygon <lat> <long> <radius_meter> [<tag>=<value>]...

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
