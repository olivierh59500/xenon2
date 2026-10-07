# Local asset setup

Original disk images and extracted artwork are not distributed in this repository.
The import and export tools rebuild them locally from a compatible Xenon 2:
Megablast Amiga ADF. The game does not execute the disk's program.

## Supported disk revision

The initial exporter supports the single-disk AmigaDOS revision identified as
`[cr BS1][h Black Monks][t +36 Black Monks]`. Its SHA-256 is
`e06a9c6fa2ab8173115f797b46fe26c9b5098ac402e8c2346175e863212b3195`.
Individual resources are verified separately, so a disk with identical game
payloads and different filesystem metadata can also be accepted. Other releases
are rejected explicitly; their code and data offsets must be verified first.

```sh
./scripts/prepare-assets.sh -adf "/path/to/Xenon 2.adf"
```

To check a disk without writing files:

```sh
./scripts/prepare-assets.sh -adf "/path/to/Xenon 2.adf" -dry-run
```

To verify the already imported resources:

```sh
./scripts/prepare-assets.sh -verify
```

## Resource directories

`assets/original/` contains the six packed resource containers. Five correspond
to gameplay levels; the sixth is the equipment shop. `.local/imported/` contains
their decoded analysis images and the decoded main executable. These decoded
images combine program code and artwork and are confined to offline analysis.

`assets/runtime/` contains the resources used by the Go game: PNG images and
ordinary JSON data. Exported terrain uses stable tile IDs, five 20 × 300 maps,
masked tile atlases, 320 × 192 backgrounds and the original palettes. The common
font retains its 38-character order and 16 × 16 dimensions. The player's ship
retains all five banking images, its thirteen steering lookup positions and
the exact positioning anchors and collision boxes. Common actor artwork and
equipment previews and ordinary shot tiers are collected in a shared named atlas. The shop catalogue
contains all 25 original English item names, prices and looping preview images.

Moving actor resources retain linked parts, animation durations, collision boxes,
scores and difficulty flags. Level-specific heading and entry-edge image choices
are represented as named metadata. Path resources use the recovered signed sine
table and resolve jumps into command indices. Encounter resources preserve source
order and contain the complete moving and fixed placement streams.

Audio resources are exported separately by `cmd/export-audio`. They include
signed eight-bit PCM samples, the recovered score, sample periods and named
effect-envelope data. The preparation script runs gameplay, shop and presentation
exporters, plus the separate gameplay and shop audio exporters.
Verified fixed-tile graphics are exported for cannons and gates in level one,
cannons and hatches in level two, small cannons in levels three and five, and
cannons in level four. Each frame includes stable tile IDs, dimensions and
original destroyed-state artwork. Ordinary fixed sprite animations are also
available for each level. Additional compound bosses, gates and special scenery
remain subject to the comparisons described in the native-data audit.

All three directories are excluded locally from Git, except explanatory marker
files. The original disk, recovered programs, executable addresses and CPU state
are not bundled with the production game.

The preparation script installs these exclusions before writing resources. It
then imports the original containers and runs the graphics/data exporter. The
`-dry-run` and `-verify` modes do not overwrite exported game resources. To rebuild
only the exports after an exporter update:

```sh
./scripts/prepare-assets.sh -adf "/path/to/Xenon 2.adf"
```

## Disk inspection

```sh
GOWORK=off go run ./cmd/disk-inspect -adf "/path/to/Xenon 2.adf"
```

The inspector prints reachable OFS/FFS files, sizes and hashes as JSON. The
optional `-extract .local/disk-files -unpack` arguments recover all files and the
main packed executable for local comparison without launching any disk content.
