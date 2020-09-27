These are some brief notes on how I set up the PostreSQL database with
osm data.

See also:
- https://github.com/openstreetmap/osm2pgsql#usage
- https://www.volkerschatz.com/net/osm/osm2pgsql-db.html (partially deprecated)

# Dependencies
```text
postgresql-server
postgis
osm2pgsql
```

# Setting up PostreSQL
```bash
initdb -D data
postgres -D data
createdb gis
psql -d gis -c 'CREATE EXTENSION postgis;'
```

# Transforming data
Schema reference: https://wiki.openstreetmap.org/wiki/Osm2pgsql/schema

```bash
# Download the osm data:
wget 'https://download.geofabrik.de/europe/azores-latest.osm.pbf'
wget 'https://download.geofabrik.de/europe/serbia-latest.osm.pbf'

# Takes ~30s; turns the ~10MB into ~300MB:
osm2pgsql --create --database gis azores-latest.osm.pbf

# Takes ~15min; turns the ~100MB into ~3.9GB:
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

# Example queries
Finding cinemas in Belgrade:
```sql
SELECT osm_id, name FROM planet_osm_point
WHERE amenity = 'cinema'
AND way && ST_Transform(ST_GeomFromText('POLYGON((20.3 44.9, 20.3 44.7, 20.6 44.7, 20.6 44.9, 20.3 44.9))', 4326), 3857);
```

Finding bicycle shops with coordinates in a part of Bremen:
```sql
SELECT osm_id, name, ST_AsText(ST_Transform(way, 4326)) FROM planet_osm_point
WHERE shop = 'bicycle'
AND way && ST_Transform(ST_GeomFromText('POLYGON((8.7968 53.1037, 8.8142 53.1037, 8.8142 53.0834, 8.7968 53.0834, 8.7968 53.1037))', 4326), 3857);
```
