#!/bin/sh
# Makes the home page teaser (EP-84) in web/static/teaser from a running
# Axon with the demo on: ./scripts/teaser/make-teaser.sh [base URL].
# Needs Docker; the Playwright image matches web/package.json.
set -eu

BASE=${1:-http://localhost:3000}
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
OUT=$ROOT/web/static/teaser
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT
USER_ID="$(id -u):$(id -g)"
mkdir -p "$OUT"

# The music generator, bundled for the browser that renders the soundtrack.
docker run --rm -u "$USER_ID" -e HOME=/tmp -v "$ROOT:/repo" -v "$WORK:/work" -w /repo/web \
	node:24-alpine npx esbuild src/lib/music.ts --bundle --format=iife --global-name=AxonMusic --outfile=/work/music.js

cp "$ROOT/scripts/teaser/record.mjs" "$WORK/"
docker run --rm --network host --ipc host -u "$USER_ID" -e HOME=/tmp \
	-v "$WORK:/work" -v "$ROOT/web/node_modules:/work/node_modules:ro" -v "$OUT:/out" -w /work \
	mcr.microsoft.com/playwright:v1.63.0-noble node record.mjs "$BASE" /work /out

DURATION=$(cat "$WORK/duration")
FADE=$(awk "BEGIN { print $DURATION - 2.5 }")
FILTER="[0:v]fps=30,scale=out_range=limited,format=yuv420p[v];[1:a]afade=t=in:d=1.5,afade=t=out:st=$FADE:d=2.5[a]"
encode() {
	docker run --rm -u "$USER_ID" -v "$WORK:/work" -v "$OUT:/out" -w /work jrottenberg/ffmpeg:7.1-alpine \
		-hide_banner -loglevel error -y -f concat -safe 0 -i frames.txt -i music.wav \
		-filter_complex "$FILTER" -map '[v]' -map '[a]' -t "$DURATION" "$@"
}
# VP9 and Opus, listed first: smaller, and for browsers without H.264 (some Linux builds of Chromium).
encode -c:v libvpx-vp9 -crf 40 -b:v 0 -row-mt 1 -deadline good -c:a libopus -b:a 80k /out/axon.webm
# H.264 and AAC for Safari; faststart lets it play while it downloads.
encode -c:v libx264 -preset veryslow -crf 30 -profile:v high -c:a aac -b:a 96k -movflags +faststart /out/axon.mp4
# The poster: the dashboard, a few seconds into the tour.
docker run --rm -u "$USER_ID" -v "$OUT:/out" jrottenberg/ffmpeg:7.1-alpine \
	-hide_banner -loglevel error -y -ss 6 -i /out/axon.mp4 -frames:v 1 -q:v 4 /out/axon.jpg
ls -l "$OUT"
