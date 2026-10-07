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
GOWORK=off go run ./cmd/export-assets
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
retains its 32 × 27 image and positioning anchor. The shop catalogue contains
the original English item names and prices.

All three directories are excluded locally from Git, except explanatory marker
files. The original disk, recovered programs, executable addresses and CPU state
are not bundled with the production game.

## Disk inspection

```sh
GOWORK=off go run ./cmd/disk-inspect -adf "/path/to/Xenon 2.adf"
```

The inspector prints reachable OFS/FFS files, sizes and hashes as JSON. The
optional `-extract .local/disk-files -unpack` arguments recover all files and the
main packed executable for local comparison without launching any disk content.
