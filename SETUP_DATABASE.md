These are some brief notes on how I set up the PostreSQL database with
osm data.

See also:
- https://github.com/openstreetmap/osm2pgsql#usage
- https://www.volkerschatz.com/net/osm/osm2pgsql-db.html

# Dependencies
```text
postgresql-server
postgis
osm2pgsql
```

# Setting up PostreSQL
```bash
initdb -D data

# Start the database server; leave this running and contiue in another
# terminal:
postgres -D data

createdb gis
psql -d gis -c 'CREATE EXTENSION postgis;'
```

# Filling the database with osm data
Use one of the following two code blocks to get started; the examles
in the README file works for `bremen-latest.osm.pbf`. When everything
worked out, you can start playing around with bigger `*.pbf` files,
like Sweden, the USA or even the whole planet. You can find those
at [download.geofabrik.de](https://download.geofabrik.de) and
[planet.osm.org](https://planet.osm.org).

Bremen, Germany (nice and small; good for quick testing):
```bash
wget 'https://download.geofabrik.de/europe/germany/bremen-latest.osm.pbf'

# Takes ~40s; turns the ~16.5MB *.pbf file into a ~265MB database:
osm2pgsql --create --database gis bremen-latest.osm.pbf
```

Serbia (a little bigger, but still good for testing on a simple laptop):
```bash
wget 'https://download.geofabrik.de/europe/serbia-latest.osm.pbf'

# Takes ~15min; turns the ~100MB *.pbf file into a ~3.9GB database:
# --number-processes 1 seems to be necessary since max_connections in
# data/postgresql.conf is limited to 20 on OpenBSD.
osm2pgsql \
	--create \
	--database gis \
	--slim \
	-C 1000 \
	--number-processes 1 \
	serbia-latest.osm.pbf
```

# Example queries (you can skip this)
To make sure everything works, you can use queries like the following
ones. Enter these into a PostreSQL shell. Such a shell can be opened
with `psql -d gis`. Make sure the PostreSQL server is running
(`postgres -d data`).

Finding bicycle shops with coordinates in a part of Bremen:
```sql
SELECT osm_id, name, ST_AsText(ST_Transform(way, 4326)) FROM planet_osm_point
WHERE shop = 'bicycle'
AND way && ST_Transform(ST_GeomFromText('POLYGON((8.7968 53.1037, 8.8142 53.1037, 8.8142 53.0834, 8.7968 53.0834, 8.7968 53.1037))', 4326), 3857);
```

Finding cinemas in Belgrade:
```sql
SELECT osm_id, name FROM planet_osm_point
WHERE amenity = 'cinema'
AND way && ST_Transform(ST_GeomFromText('POLYGON((20.3 44.9, 20.3 44.7, 20.6 44.7, 20.6 44.9, 20.3 44.9))', 4326), 3857);
```

Schema reference: https://wiki.openstreetmap.org/wiki/Osm2pgsql/schema
