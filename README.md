# OpenKye level index

`index.json` lists every known Kye level set that can be downloaded from the
internet, in the style of a lockfile: each set is identified by the SHA-256 of
its file and lists the URLs it can be fetched from. The files themselves are
not stored here.

`kye-server -level-index https://raw.githubusercontent.com/openkye/level-index/main/index.json`
downloads this index on startup and imports every set its database does not have yet, verifying each download
against the set's `sha256` after normalizing it.

## Format

```json
{
  "version": 1,
  "sets": [
    {
      "filename": "DEFAULT.KYE",
      "author": "Colin Garbutt",
      "homepage": "https://example.com/kye/",
      "sha256": "…",
      "levels": 30,
      "sources": [
        { "url": "https://example.com/kye.zip!/KYE/DEFAULT.KYE", "sha256": "…" },
        { "url": "https://example.org/levels/default.kye" }
      ]
    }
  ]
}
```

- `filename`: the first known name of the file.
- `author`: who made the levels, omitted when unknown.
- `homepage`: the set's home page, omitted when unknown. Where a set is linked
  from several pages, this is a best effort at the canonical one.
- `sha256`: lowercase hex SHA-256 of the normalized file: the file parsed and
  re-encoded with LF line endings, trimmed name/hint/message lines, nothing
  after the last level, and each board as the engine plays it
  (`kye.Board.Normalize`): short rows padded with spaces, border cells that
  are not walls turned into wall `5`, only the last `K` kept, and a `K` added
  at (3, 3) when there is none. Boards with Kye 3.0 teleporters keep their
  border and Kyes, since Kye 3.0 wraps around border gaps. Copies that differ
  only in these ways, such as a DOS copy with a trailing Ctrl-Z, are one set
  with several sources. SKye's TI-89 level libraries (`.89y`) are not
  converted yet: their `sha256` is of the exact file, `levels` comes from the
  library header, and they are always marked `skye`, so the server does not
  import them.
- `levels`: the number of levels in the file.
- `extensions`: game features beyond Kye 2.0 that the set needs, omitted when
  it needs none. `kye3` is Kye 3.0: teleporters `(` `)` `_` `'` facing right,
  left, up and down, and gaps in the border wall that wrap around the board.
  `skye` is SKye: pits `O`, bombs `!` and sticky blocks `P` `p`. Classic Kye
  ignores these characters and fills border gaps with wall, and some classic
  levels contain them by accident, so `kye-index add` detects extensions only
  when it creates a set, warns about each one, and leaves a set's extensions
  alone afterwards; remove a wrong detection by hand, or add one with
  `-extensions kye3`. `unknown` marks a set with a level that no known version
  plays as written: a character no version defines, no Kye or more than one,
  or no diamonds.
- `sources`: every known download, best first. `url` downloads the file
  directly; `!/` separates an archive (zip, tar, tar.gz, or an LHA .lzh or
  self-extracting .exe using -lh0- or -lh5-) from the path of a file inside
  it, and archives can nest. A file compressed with MS-COMPRESS (SZDD, as in
  `DEFAULT.KY_`) is expanded. Only sources these formats can decode are
  listed, unless a set has no other. Dead sites are referenced through the
  Wayback Machine's raw form,
  `https://web.archive.org/web/<timestamp>id_/<original URL>`. A source has
  its own `sha256`, of the exact downloaded bytes (after MS-COMPRESS
  expansion), only when they differ from the normalized file.

Sets are sorted by filename, then `sha256`.

## Adding a level set

Use `kye-index` from the [kye repository](https://github.com/openkye/kye):

```sh
go run github.com/openkye/kye/cmd/kye-index add \
  -homepage https://example.com/downloads.html -author "Jane Doe" \
  'https://example.com/levels.zip!/JANE.KYE'
```

It downloads the URL, checks that it is a Kye level file, and adds it as a
source of the set with the same normalized `sha256`, creating the set if it is
new. `-homepage` and `-author` fill in a set's fields only when they are
empty. Check that every source still matches with:

```sh
go run github.com/openkye/kye/cmd/kye-index verify
```
