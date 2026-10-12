#!/bin/sh
# Imports an Axon-generated Anki package with the official Anki library, so
# "exported decks import into Anki without errors" stays tested (EP-16).
set -eu
dir=$(mktemp -d)
trap 'rm -rf "$dir"' EXIT
chmod 777 "$dir"
(cd api && AXON_APKG_OUT="$dir/sample.apkg" go test ./internal/cards/ -run TestAPKGForAnki -count=1 >/dev/null)
chmod 644 "$dir/sample.apkg"
cat > "$dir/check.py" <<'PY'
from anki.collection import Collection
from anki.import_export_pb2 import ImportAnkiPackageRequest, ImportAnkiPackageOptions
col = Collection("/tmp/check.anki2")
out = col.import_anki_package(ImportAnkiPackageRequest(package_path="/data/sample.apkg", options=ImportAnkiPackageOptions(with_scheduling=False)))
assert len(out.log.new) == 2, out.log
assert {"Learning science", "Rust"} <= {d.name for d in col.decks.all_names_and_ids()}
assert len(__import__("os").listdir(col.media.dir())) == 1
print("Anki imported", len(out.log.new), "notes")
PY
docker run --rm --network host -v "$dir:/data" python:3.12-slim sh -c 'pip install -q --root-user-action=ignore "anki==26.9.3" && python /data/check.py'
