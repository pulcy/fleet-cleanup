# Fleet-cleanup

Fleet is a nice distributed init system, but has a nasty habbit of leaving garbage around in ETCD.
This utility removes that garbage.

## Usage

```
docker run -it --rm --net=host pulcy/fleet-cleanup:latest [--dry-run]
```
