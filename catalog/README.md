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

## Tap signing (for third-party catalog maintainers)

Third-party catalogs (taps) can be cryptographically signed with minisign so
users can pin trust to a public key on top of TOFU. Layout at the tap repo
root:

```
CATALOG.checksum    # deterministic listing of catalog files
CATALOG.minisig     # minisign signature of CATALOG.checksum
<service-id>/
  service.yaml
  ...
```

`CATALOG.checksum` lines have the form `<sha256>  <relative/path>`, sorted
lexicographically by path, covering every file under the repo root except
the two signature files and anything below dot-prefixed directories
(`.git`, `.github`, etc.).

Maintainer one-time setup:

```
minisign -G                                   # generate keypair
cat minisign.pub                              # publish this line in your README
```

After every catalog change:

```
find . -type f \
  ! -name 'CATALOG.checksum' ! -name 'CATALOG.minisig' \
  ! -path './.*' \
  -print0 | sort -z | xargs -0 sha256sum > CATALOG.checksum
minisign -Sm CATALOG.checksum                 # produces CATALOG.minisig
git add CATALOG.checksum CATALOG.minisig
git commit -m "catalog: re-sign"
```

Users opt in to verification when adding the tap:

```
one tap add user/repo --verify-key 'RWQ…<published-key>'
# or
one tap add user/repo --verify-key-file ./tap.pub
```

The pinned public key is persisted in `~/.one/taps/registry.json`. Every
subsequent `one tap update` re-verifies the catalog against that key; if
the signature breaks (key revoked, file tampered, file added but unsigned),
the update aborts and the pinned SHA is unchanged.

WASM handlers from taps stay refused by default regardless of signature
(see `ONE_TAP_ALLOW_HANDLERS`). Signature proves authorship, not safety.
