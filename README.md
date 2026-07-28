# JSON Really Pretty (jrp) CLI

A JSON formatter that lays out a document by the *shape* of the data instead of
putting every single value on its own line.

The Golang formatting engine lives in the [json-really-pretty-go](https://github.com/patrickbergner/json-really-pretty-go)
library. This repo is the CLI built on top of it.

Besides the Golang library, there are also ports
for Java at [json-really-pretty-java](https://github.com/patrickbergner/json-really-pretty-java)
and PHP at [json-really-pretty-php](https://github.com/patrickbergner/json-really-pretty-php).

## Why this exists

Every common pretty printer treats JSON as a list of members: one member, one
line. That is fine for a small nested config, and terrible for everything else.
A file of 200 flat records turns into 1400 lines of scroll, and a value you
could have read at a glance is smeared over four lines:

```json
{
    "id": 1,
    "name": "alpha",
    "scores": [
        10,
        20,
        30
    ],
    "active": true
}
```

Compact output (`jq -c`, `JSON.stringify`) has the opposite problem: it fits on
one line, but nothing is readable.

`jrp` picks the middle. The rule is a single one:

> A container is written on one line **unless it holds an object somewhere
> inside it.**

So a plain record stays a single line however long it gets, while anything with
real structure below it opens up and shows that structure. The same data:

```json
[
    { "id": 1, "name": "alpha", "scores": [10, 20, 30], "active": true },
    { "id": 2, "name": "beta", "scores": [], "active": false },
    { "id": 3, "name": "gamma", "scores": [7], "active": null }
]
```

The result reads like a table where the data is tabular, and like a tree where
the data is a tree — which is normally what you wanted to see in the first
place.

A nested config behaves the same way. Leaf groups collapse, the structure that
carries them stays open:

```json
{
    "service": {
        "name": "gateway",
        "port": 8080,
        "tls": { "enabled": true, "ciphers": ["TLS_AES_128_GCM_SHA256", "TLS_AES_256_GCM_SHA384"] }
    },
    "replicas": 3,
    "hosts": ["a.example.com", "b.example.com", "c.example.com"],
    "labels": { "tier": "edge" },
    "flags": [],
    "limits": { "cpu": "500m", "mem": "512Mi" }
}
```

Two more properties worth knowing:

- **The rewrite is textual, not a round trip through a decoded value.** Key
  order and number formatting survive exactly as written — no `1e3` turning
  into `1000`, no reordered members, no lost precision on big integers.
- **In-place rewrites are atomic.** `-w` writes a temporary file next to the
  original and renames it over, carrying the original file mode, so an
  interrupted run cannot leave a truncated file behind.

## Usage

```
jrp [flags] [file ...]
```

Reads from stdin when given no files or `-`, and writes to stdout unless
`-w`/`--write` is set. Multiple files may be given at once.

```bash
jrp config.json                 # format to stdout
curl -s https://api/x | jrp     # format a response
jrp -w src/**/*.json            # rewrite files in place
jrp -x 100 -n 12 data.json      # with hard caps (see below)
```

Exit status is `0` on success and `1` if any input could not be read or parsed;
a file that fails to parse is reported on stderr and left untouched.

## Options

| Flag | Default | Description |
| --- | --- | --- |
| `-w`, `--write` | off | Rewrite each named file in place instead of writing to stdout. Cannot be combined with stdin. |
| `-i`, `--indent int` | `4` | Number of spaces per indentation level. |
| `-t`, `--tab` | off | Indent with tabs instead of spaces. |
| `-x`, `--max-width int` | `0` | Expand containers whose one-line form would pass this column. `0` disables. |
| `-n`, `--max-items int` | `0` | Expand containers holding more than this many members or elements. `0` disables. |
| `-b`, `--tight-braces` | off | Write inline objects as `{"a": 1}` instead of `{ "a": 1 }`. |
| `-p`, `--padded-brackets` | off | Write inline arrays as `[ 1, 2 ]` instead of `[1, 2]`. |
| `-s`, `--sort-keys` | off | Sort object keys instead of keeping the input order. |
| `-v`, `--version` | | Print the version and exit. |
| `-h`, `--help` | | Show the help message. |

### The two caps

The structural rule above is always in force and is not configurable — it is
what the tool is for. `--max-width` and `--max-items` sit *on top* of it as
optional ceilings for people who do want a limit, and both are off by default.

- `-x 40` expands anything whose single line would reach past column 40, so a
  long record falls back to conventional one-member-per-line output while its
  short neighbours stay inline. The cap is best effort: a single string literal
  longer than the limit still overflows, because JSON strings cannot be wrapped.
- `-n 3` expands anything with more than three members or elements, regardless
  of how short the line would be.

Both are measured over the whole subtree, so a container that is approved for
one line keeps everything below it on that line too.

### Style options

`--tight-braces` and `--padded-brackets` only affect *inline* containers.
The defaults (`{ "a": 1 }` and `[1, 2]`) are chosen so that objects and arrays
stay visually distinct at a glance even when both are on the same line.

`--sort-keys` is the only flag that changes the data's presentation order;
without it the input order is preserved byte for byte.

## Building

Requires Go 1.24 or newer.

```bash
go build -o jrp .
```

`build.sh` formats, vets, tests and cross compiles static binaries for
`windows/amd64`, `windows/arm64`, `linux/amd64`, `linux/arm64` and
`darwin/arm64` into `dist/`:

```bash
./build.sh
```
