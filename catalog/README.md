# Official catalog

Services bundled into the `one` binary at build time via `//go:embed`.

Layout:

```
catalog/
  services/
    <service-id>/
      service.yaml
      SKILL.md                (optional)
      actions/<id>.yaml       (optional, per-action files)
      guides/<id>.md          (optional)
```

The chain resolution order is **embed → local FS (`$HOME/.one/catalog`) → HTTP (if `ONE_CATALOG_URL` set) → taps**. Embed wins on conflict.

To add a service: create a directory under `services/`, validate with `one catalog lint` (against this root), commit, rebuild. The next release ships it.

CI must run `make test` to verify all embedded services parse.
