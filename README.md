# Xenon 2 Go

An independent Go/Ebitengine recreation of Xenon 2: Megablast for the Amiga.
The original disk is used only by local extraction and comparison tools.
The game engine will use ordinary Go state and exported audiovisual data;
it will not execute the original program or load its code as game logic.

The conversion is in development. Disk reading and validated decompression are
available; level behavior, graphics, weapons, shops and timing are being analyzed.
The complete game is not yet playable.

## Local resources

Supply a compatible original Amiga ADF in a local directory. Original disks,
compressed containers, recovered programs and generated assets are not included
in Git. Extraction tools rebuild the local resources reproducibly.

```sh
./scripts/prepare-assets.sh -adf "/path/to/Xenon 2.adf"
```

See [reference notes](docs/REFERENCE_NOTES.md) and the
[native-data audit](docs/NATIVE_AUDIT.md) for verified findings and comparison
boundaries. The supplied disk is a trained revision; trainer behavior is kept
separate from the original game's rules.
