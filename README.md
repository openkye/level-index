# Kye Level Index

This repository is intended to become an index of every known public Kye level
set in the classic text `.kye` format.

The canonical index is [`index.json`](index.json). It records the SHA-256 hash
of each exact `.kye` file and one or more public URLs where those same bytes can
be obtained. The hash is deliberately for the extracted `.kye` file, not for a
`.zip` wrapper, installer, web page, or archive item.

## Index format

The current format is JSON, versioned by `schemaVersion`. The JSON Schema is in
[`index.schema.json`](index.schema.json).

Top-level fields:

- `schemaVersion`: integer format version. Currently `1`.
- `generatedAt`: RFC 3339 timestamp for the current index revision.
- `description`: human-readable notes about the index.
- `sources`: optional list of public catalogues or collections used while
  researching entries.
- `levelSets`: sorted list of indexed `.kye` files.

Each `levelSets` entry contains:

- `id`: stable repository-local identifier, normally based on the filename and
  author if needed to avoid collisions.
- `title`: display name of the level set.
- `author`: optional creator or credited author.
- `sha256`: lowercase SHA-256 of the exact `.kye` bytes.
- `bytes`: optional byte length of the exact `.kye` file.
- `levelCount`: optional number of levels in the set, when known.
- `fileName`: optional original `.kye` filename.
- `description`: optional short description from the author or source page.
- `license`: optional license, terms, or shareware notice.
- `sourceIds`: optional references to top-level `sources` used for this entry.
- `urls`: one or more locations for the exact file or an archive containing it.
- `notes`: optional maintainer notes.

A minimal entry looks like this:

```json
{
  "id": "quests",
  "title": "Quests",
  "author": "Colin Phipps",
  "sha256": "0000000000000000000000000000000000000000000000000000000000000000",
  "fileName": "quests.kye",
  "urls": [
    {
      "url": "https://games.moria.org.uk/kye/quests.kye",
      "sourceId": "python-kye-my-levels"
    }
  ]
}
```

Do not commit placeholder hashes in real entries. If the file cannot be fetched
and hashed, leave it out until the exact bytes are available.

## Known public sources to harvest

The seed source list currently points to public Kye catalogues that should be
harvested into concrete `levelSets` entries once their `.kye` bytes are
available:

- My Kye Page level sets: <https://kye.me.uk/levels.html>
- Python Kye / Colin Phipps levels: <https://games.moria.org.uk/kye/mylevels>
- My Kye Page charity levels: <https://kye.me.uk/charitylevels.html>

## Tooling

The Go command under `cmd/kyeindex` validates repository-specific invariants and
computes file hashes.

```sh
go run ./cmd/kyeindex validate
go run ./cmd/kyeindex hash path/to/file.kye
```

Validation checks include:

- JSON decoding with unknown-field rejection.
- schema version and RFC 3339 timestamps.
- stable identifier syntax and uniqueness.
- lowercase SHA-256 syntax and uniqueness.
- `.kye` filename suffixes.
- URL syntax and source references.
- `levelSets` sorted by `id`.
