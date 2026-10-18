#!/bin/sh
# Makes Axon's trailer (EP-125) from a running Axon with the demo on:
# ./scripts/trailer/make-trailer.sh [base URL]. Screenshots of the demo, a
# film drawn frame by frame in 4K at 60 fps (trailer.html), and a soundtrack
# composed for it, mastered to -14 LUFS. Writes the 4K master to
# dist/axon-trailer-4k.mp4 (a release asset, too big for the repository) and
# the home page's 1080p copies to web/static/teaser. Needs Docker; the
# Playwright image matches web/package.json.
set -eu

BASE=${1:-http://localhost:3000}
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
OUT=$ROOT/web/static/teaser
DIST=$ROOT/dist
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT
USER_ID="$(id -u):$(id -g)"
PW=mcr.microsoft.com/playwright:v1.63.0-noble
FF=jrottenberg/ffmpeg:7.1-alpine
mkdir -p "$OUT" "$DIST" "$WORK/shots"
cp -r "$ROOT/scripts/trailer/trailer.html" "$ROOT/scripts/trailer/render.mjs" "$ROOT/scripts/trailer/shots.mjs" "$ROOT/scripts/trailer/fonts" "$WORK/"

run() {
	docker run --rm --network host --ipc host -u "$USER_ID" -e HOME=/tmp \
		-v "$WORK:$WORK" -v "$ROOT/web/node_modules:$WORK/node_modules:ro" -w "$WORK" $PW node "$@"
}
run shots.mjs "$BASE" "$WORK/shots"
run render.mjs "$WORK"

DURATION=$(cat "$WORK/duration")
FADE=$(awk "BEGIN { print $DURATION - 2.5 }")
AUDIO="[1:a]afade=t=in:d=0.3,afade=t=out:st=$FADE:d=2.5,loudnorm=I=-14:TP=-1:LRA=11,aresample=48000[a]"
ff() { docker run --rm -u "$USER_ID" -v "$WORK:/work" -v "$OUT:/out" -v "$DIST:/dist" -w /work $FF -hide_banner -loglevel error -y "$@"; }
input="-framerate 60 -i frames/%05d.jpg -i music.wav"

# The master: 4K, 60 fps, H.264 High at a quality that keeps text crisp.
# shellcheck disable=SC2086 # $input is a list of arguments
ff $input -filter_complex "[0:v]format=yuv420p[v];$AUDIO" -map '[v]' -map '[a]' -t "$DURATION" \
	-c:v libx264 -preset slow -crf 17 -profile:v high -level 5.2 -tune animation \
	-c:a aac -b:a 256k -movflags +faststart /dist/axon-trailer-4k.mp4
# The home page's copies: 1080p60, scaled down from 4K. VP9 and Opus first, for
# browsers without H.264; then H.264 and AAC for Safari, playing as it loads.
WEB="[0:v]scale=1920:1080:flags=lanczos,format=yuv420p[v];$AUDIO"
# shellcheck disable=SC2086
ff $input -filter_complex "$WEB" -map '[v]' -map '[a]' -t "$DURATION" \
	-c:v libvpx-vp9 -crf 31 -b:v 0 -row-mt 1 -deadline good -c:a libopus -b:a 160k /out/axon.webm
# shellcheck disable=SC2086
ff $input -filter_complex "$WEB" -map '[v]' -map '[a]' -t "$DURATION" \
	-c:v libx264 -preset slow -crf 19 -profile:v high -tune animation -c:a aac -b:a 192k -movflags +faststart /out/axon.mp4
# The poster: the name, once the mark has drawn itself.
ff -i frames/00380.jpg -vf scale=1920:1080:flags=lanczos -q:v 2 /out/axon.jpg
cp "$WORK/axon.en.vtt" "$WORK/axon.fr.vtt" "$OUT/"
ls -l "$OUT" "$DIST"
