# domain-list-community (fork)

**English** | [简体中文](README.zh.md)

A minimal repackaging fork of
[`v2fly/domain-list-community`](https://github.com/v2fly/domain-list-community).
It does not maintain its own domain data; it rebuilds a trimmed `dlc.dat` that
contains exactly the data required by two sing-box rule-sets in
[`yagh779/sing-geosite`](https://github.com/yagh779/sing-geosite):

| Entry | Content | Feeds tag |
| --- | --- | --- |
| `cn` | Overlay domains tagged `@!cn` (from `data/cn-overseas`) | `cn@!cn` |
| `geolocation-!cn` | Upstream `geolocation-!cn` domains tagged `@cn` only | `geolocation-!cn@cn` |

Only `dlc.dat` and `dlc.dat.sha256sum` are published, on the `release` branch.

## Data sources

- `geolocation-!cn@cn`: the latest `dlc.dat` from [`v2fly/domain-list-community`](https://github.com/v2fly/domain-list-community).
- `cn@!cn`: the latest `dlc.dat` from [`nekolsd/domain-list-community`](https://github.com/nekolsd/domain-list-community),
  materialized into [`data/cn-overseas`](data/cn-overseas) as `@!cn`-tagged rules.

## Build pipeline

The `Build dlc.dat` workflow ([.github/workflows/build.yml](.github/workflows/build.yml))
runs on a daily schedule (`30 20 * * *` UTC) and on manual dispatch:

1. Download the latest `dlc.dat` from both source repositories.
2. Compute a stable fingerprint of each target tag (`cn@!cn`, `geolocation-!cn@cn`)
   and compare it with `sync-state.json`.
3. Stop without building when both fingerprints are unchanged (unless a manual run
   forces a rebuild).
4. Refresh `data/cn-overseas` when `cn@!cn` changed.
5. Repackage a trimmed `dlc.dat` (only the domains above) and push it with its
   checksum to the `release` branch.
6. Commit the updated `sync-state.json` back to `master`.

`sync-state.json` stores both fingerprints under their tag names:

```json
{
  "cn@!cn": "<sha256>",
  "geolocation-!cn@cn": "<sha256>"
}
```

## Local usage

```sh
# fingerprints
go run ./cmd/sync hashtag <dlc.dat> 'cn@!cn'
go run ./cmd/sync hashtag <dlc.dat> 'geolocation-!cn@cn'

# refresh the cn@!cn overlay from a nekolsd dlc.dat
go run ./cmd/sync extract-cn-overseas <nekolsd-dlc.dat> data/cn-overseas

# repackage the trimmed dlc.dat
go run ./cmd/build <v2fly-dlc.dat> data/cn-overseas dist/dlc.dat
```

## Repository layout

| Path | Purpose |
| --- | --- |
| `cmd/sync/` | Fingerprint and overlay extraction helpers |
| `cmd/build/` | Repackages the trimmed `dlc.dat` |
| `data/cn-overseas` | `@!cn`-tagged overlay feeding `cn@!cn` |
| `sync-state.json` | Last-built tag fingerprints |

Domain rule syntax (`domain:`, `full:`, `keyword:`, `regexp:`, `@attr`, `include:`)
follows upstream; see the
[upstream README](https://github.com/v2fly/domain-list-community#structure-of-data)
for details.
