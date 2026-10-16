#!/bin/sh
# Capture five minutes of expert play and add English captions below the game.
set -eu
project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$project_root"
prefix=${XENON2_PRESENTATION_PREFIX:-recordings/xenon2-presentation-5min}
source_movie=${XENON2_PRESENTATION_SOURCE:-$prefix-source.mp4}
export GOWORK=off
for extension in mp4 webm; do
    if [ -e "$prefix.$extension" ]; then
        echo "Output exists: $prefix.$extension. Select another XENON2_PRESENTATION_PREFIX." >&2
        exit 1
    fi
done
if [ ! -f "$source_movie" ]; then
    go run ./cmd/video -duration 5m -output "$source_movie" -poster-at 75s
fi
go run ./tools/presentation -output "$prefix-captions"
cp "$prefix-captions/presentation.en.srt" "$prefix.en.srt"
ffmpeg -hide_banner -loglevel error -nostdin -n -i "$source_movie" \
    -f concat -safe 0 -i "$prefix-captions/captions.ffconcat" \
    -filter_complex '[0:v]pad=1280:896:0:0:color=0x0b131d[game];[game][1:v]overlay=0:800:shortest=1:eof_action=repeat,fps=60,tpad=stop_mode=clone:stop_duration=1[video]' \
    -map '[video]' -map 0:a:0 -map_metadata 0 -map_chapters 0 \
    -c:v libx264 -preset medium -crf 19 -threads 4 -pix_fmt yuv420p \
    -c:a copy -t 300 -frames:v 18000 -movflags +faststart "$prefix.mp4"
ffmpeg -hide_banner -loglevel error -nostdin -n -i "$prefix.mp4" \
    -vf 'scale=640:448:flags=area' -c:v libvpx-vp9 -b:v 550k -crf 36 \
    -maxrate 850k -bufsize 1700k -row-mt 1 -cpu-used 3 -threads 4 -pix_fmt yuv420p \
    -c:a libopus -b:a 96k -t 300 -frames:v 18000 "$prefix.webm"
echo "Created $prefix.mp4, $prefix.webm and $prefix.en.srt"
