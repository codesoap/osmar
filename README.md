A small tool for querying osm data from PBF files.

# Installation
```bash
go install github.com/codesoap/osmar/v3@latest
# The binary is now at ~/go/bin/osmar.
```

# Basic Usage
Before you can use osmar, you must set the environment variable
`OSMAR_PBF_FILE`; it must contain the path to a PBF
file. These files can be downloaded, for example, from
[download.geofabrik.de](https://download.geofabrik.de/). An example would be:
```console
$ cd /tmp
$ wget 'https://download.geofabrik.de/europe/germany/bremen-latest.osm.pbf'
$ export OSMAR_PBF_FILE='/tmp/bremen-latest.osm.pbf'
```

```console
$ # Find all entries within 50m of the center of Bremen, Germany:
$ osmar 53.076 8.807 50
meta:distance: 5m
meta:id: 163740903
meta:type: way
meta:link: https://www.openstreetmap.org/way/163740903
addr:city: Bremen
addr:country: DE
addr:housenumber: 1
addr:postcode: 28195
addr:street: Am Markt
building: retail
...

$ # Filter by tags to find a bicycle shop near the center of Bremen:
$ osmar 53.076 8.807 500 shop=bicycle
meta:distance: 243m
meta:id: 834082330
meta:type: node
meta:link: https://www.openstreetmap.org/node/834082330
addr:city: Bremen
addr:country: DE
addr:housenumber: 30-32
addr:postcode: 28195
addr:street: Martinistraße
check_date:opening_hours: 2024-04-28
email: velo-sport@nord-com.net
fax: +49 421 18225
name: Velo-Sport
...

$ # Use UNIX tools to compact the output:
$ osmar 53.076 8.807 200 shop=clothes | grep -e '^$' -e distance -e meta:link -e name
meta:distance: 65m
meta:link: https://www.openstreetmap.org/node/410450005
name: Peek & Cloppenburg
short_name: P&C

meta:distance: 98m
meta:link: https://www.openstreetmap.org/node/3560745513
name: CALIDA

meta:distance: 99m
meta:link: https://www.openstreetmap.org/node/718963532
name: zero
...
```

# More Examples
You can find the documentation on all available tags at
[wiki.openstreetmap.org/wiki/Map_Features](https://wiki.openstreetmap.org/wiki/Map_Features).
Here are a few more examples:

```bash
# Find a bakery:
osmar 53.076 8.807 200 shop=bakery

# Find nearby public transport stations:
osmar 53.076 8.807 200 public_transport=stop_position

# Find nearby hiking routes:
osmar 53.076 8.807 500 route=hiking

# Searching for multiple values of the same tag is also possible:
osmar 53.076 8.807 3000 sport=climbing sport=swimming

# Pro tip: Use "*" to search for any value:
osmar 53.076 8.807 500 'sport=*'

# Learn about the population of the city and its urban districts:
osmar 53.076 8.807 10000 'population=*'
```

# Performance
Because osmar is parsing compressed PBF files on the fly, performance is
somewhat limited, but should be good enough for a few queries now and
then. Try to use the smallest extract that is available for your area.

The performance can be improved slightly by converting PBF files to zstd compression with 
the [zstd-pbf tool](https://github.com/codesoap/zstd-pbf).

Here are some quick measurements; better results will probably be
achieved with more modern hardware:

| PBF file | Query | CPU | Runtime | RAM usage |
| --- | --- | --- | --- | --- |
| bremen-latest.osm.pbf (19.3MiB) | `osmar 53.076 8.807 50 'shop=*'` | i5-8250U (4x1.6GHz) | ~0.25s | ~90MiB |
| bremen-latest.osm.pbf (19.3MiB) | `osmar 53.076 8.807 50 'shop=*'` | AMD Ryzen 5 3600 (6x4.2GHz) | ~0.13s | ~150MiB |
| bremen-latest.zstd.osm.pbf (19.4MiB) | `osmar 53.076 8.807 50 'shop=*'` | i5-8250U (4x1.6GHz) | ~0.22s | ~100MiB |
| bremen-latest.zstd.osm.pbf (19.4MiB) | `osmar 53.076 8.807 50 'shop=*'` | AMD Ryzen 5 3600 (6x4.2GHz) | ~0.12s | ~150MiB |
| czech-republic-latest.osm.pbf (828MiB) | `osmar 49.743 13.379 200 'shop=*'` | i5-8250U (4x1.6GHz) | ~9s | ~400MiB |
| czech-republic-latest.osm.pbf (828MiB) | `osmar 49.743 13.379 200 'shop=*'` | AMD Ryzen 5 3600 (6x4.2GHz) | ~4.0s | ~650MiB |
| czech-republic-latest.zstd.osm.pbf (847MiB) | `osmar 49.743 13.379 200 'shop=*'` | i5-8250U (4x1.6GHz) | ~7s | ~450MiB |
| czech-republic-latest.zstd.osm.pbf (847MiB) | `osmar 49.743 13.379 200 'shop=*'` | AMD Ryzen 5 3600 (6x4.2GHz) | ~3.7s | ~675MiB |

PS: Previously osmar accessed a PostgreSQL database. This was much
faster and had some other benefits, but the database was annoying to set
up, so I abandoned this approach. You can find this version of osmar here:
[github.com/codesoap/osmar/tree/v2](https://github.com/codesoap/osmar/tree/v2)
