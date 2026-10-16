# Preparing the original game data

The Xenon 2 ADF and generated runtime resources are not distributed in this
repository. Supply your own compatible Amiga disk image and run the included Go
import/export pipeline before starting the game.

## Supported disk

The verified single-disk revision is identified as:

```text
Xenon 2 - Megablast [cr BS1][h Black Monks][t +36 Black Monks]
```

Its uncompressed ADF SHA-256 is:

```text
e06a9c6fa2ab8173115f797b46fe26c9b5098ac402e8c2346175e863212b3195
```

The [Planet Emulation Amiga ADF catalogue, letter X](https://www.planetemu.net/roms/commodore-amiga-games-adf?page=X)
lists this revision. Extract the ADF from its archive first. The two-disk
editions are different releases and are not accepted by this importer.

## Generate and verify the resources

From the repository root:

```sh
sh tools/exclude-local-assets.sh
./scripts/prepare-assets.sh -adf "/path/to/Xenon 2.adf"
./scripts/prepare-assets.sh -verify
GOWORK=off go run ./cmd/xenon2
```

The preparation script validates the disk, imports its files, decodes graphics,
menus, shop data and audio, and creates the runtime asset set. It rejects an
unsupported disk instead of reading fixed offsets from unknown data.

To verify a fresh output without replacing the active assets:

```sh
XENON2_RUNTIME_OUTPUT=.local/rebuild/runtime ./scripts/prepare-assets.sh \
  -adf "/path/to/Xenon 2.adf" \
  -output .local/rebuild/original \
  -analysis .local/rebuild/imported
```

Original disk images, imported programs and generated resources must remain
local. The helper writes exclusions to `.git/info/exclude`; no project
`.gitignore` is required.
