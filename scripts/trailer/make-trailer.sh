#!/bin/sh
# Makes Axon's trailer (EP-125) for the home page, from a running Axon with
# the demo on: ./scripts/trailer/make-trailer.sh [base URL]. Screenshots of
# the demo, a film drawn frame by frame (trailer.html), and a soundtrack
# composed for it. Needs Docker; the Playwright image matches web/package.json.
set -eu

BASE=${1:-http://localhost:3000}
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
OUT=$ROOT/web/static/teaser
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT
USER_ID="$(id -u):$(id -g)"
PW=mcr.microsoft.com/playwright:v1.63.0-noble
mkdir -p "$OUT" "$WORK/shots"
cp -r "$ROOT/scripts/trailer/trailer.html" "$ROOT/scripts/trailer/render.mjs" "$ROOT/scripts/trailer/shots.mjs" "$ROOT/scripts/trailer/fonts" "$WORK/"

run() {
	docker run --rm --network host --ipc host -u "$USER_ID" -e HOME=/tmp \
		-v "$WORK:$WORK" -v "$ROOT/web/node_modules:$WORK/node_modules:ro" -w "$WORK" $PW node "$@"
}
run shots.mjs "$BASE" "$WORK/shots"
run render.mjs "$WORK"

DURATION=$(cat "$WORK/duration")
FADE=$(awk "BEGIN { print $DURATION - 3 }")
FILTER="[0:v]scale=1280:720:flags=lanczos,fps=30,format=yuv420p[v];[1:a]afade=t=in:d=0.5,afade=t=out:st=$FADE:d=3[a]"
encode() {
	docker run --rm -u "$USER_ID" -v "$WORK:/work" -v "$OUT:/out" -w /work jrottenberg/ffmpeg:7.1-alpine \
		-hide_banner -loglevel error -y -framerate 30 -i frames/%05d.jpg -i music.wav \
		-filter_complex "$FILTER" -map '[v]' -map '[a]' -t "$DURATION" "$@"
}
# VP9 and Opus first: smaller, and for browsers without H.264.
encode -c:v libvpx-vp9 -crf 36 -b:v 0 -row-mt 1 -deadline good -c:a libopus -b:a 128k /out/axon.webm
# H.264 and AAC for Safari; faststart lets it play while it downloads.
encode -c:v libx264 -preset slow -crf 24 -profile:v high -c:a aac -b:a 160k -movflags +faststart /out/axon.mp4
# The poster: the name, and the captions.
docker run --rm -u "$USER_ID" -v "$WORK:/work" -v "$OUT:/out" jrottenberg/ffmpeg:7.1-alpine \
	-hide_banner -loglevel error -y -i /work/frames/00165.jpg -vf scale=1280:720 -q:v 3 /out/axon.jpg
cp "$WORK/axon.en.vtt" "$WORK/axon.fr.vtt" "$OUT/"
ls -l "$OUT"
