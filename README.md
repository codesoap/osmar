A small tool for playing around with osm-Databases (those resulting
from `osm2pgsql`). Read `SETUP_DATABASE.md` to learn how to set up the
required database.

Note that `osm2pgsql` by default does not put all available tags into
the database and osmf only deals with this limited tag-set.

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

# Examples
```bash
# Find all points within 100m of the Eiffel tower:
osmf point 48.85829 2.29446 100

# Find bicycle shops in Bremen:
osmf point 53.07583 8.80716 10000 shop=bicycle

# Searching for multiple values of the same tag is also possible:
osmf point 53.07583 8.80716 10000 sport=climbing sport=swimming

# Use UNIX tools to compact the output:
osmf point 48.85829 2.29446 100 | grep -e '^$' -e '^osm_id' -e '^name'
```
