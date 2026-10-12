#!/bin/sh
# End-to-end check of the Anki bridge (EP-42) against a disposable
# anki-sync-server, both from the official Anki library:
#   1. Axon's cards reach the server through the bridge;
#   2. syncing again updates notes in place (stable GUIDs), no duplicates;
#   3. a note added only in Anki survives the next bridge sync.
set -eu
net=axon-anki-check
work=$(mktemp -d)
cleanup() {
	docker rm -f axon-anki-server axon-anki-bridge >/dev/null 2>&1 || true
	docker network rm $net >/dev/null 2>&1 || true
	rm -rf "$work"
}
trap cleanup EXIT
chmod 777 "$work"

docker build -q --network "${AXON_BUILD_NETWORK:-host}" -t axon-anki:check deploy/anki >/dev/null
docker network create $net >/dev/null
docker run -d --name axon-anki-server --network $net -e SYNC_USER1=axon:secret -e SYNC_HOST=0.0.0.0 -e SYNC_PORT=8080 \
	-e SYNC_BASE=/tmp/sync axon-anki:check python -m anki.syncserver >/dev/null
docker run -d --name axon-anki-bridge --network $net -e ANKI_BRIDGE_DATA=/tmp/data axon-anki:check >/dev/null
sleep 3

(cd api && AXON_APKG_OUT="$work/sample.apkg" go test ./internal/cards/ -run TestAPKGForAnki -count=1 >/dev/null)
chmod 644 "$work/sample.apkg"

cat > "$work/client.py" <<'PY'
import base64, json, sys, urllib.request, uuid
from anki.collection import Collection
from anki.sync_pb2 import SyncCollectionResponse
R = SyncCollectionResponse.ChangesRequired
SERVER = "http://axon-anki-server:8080/"
USER = str(uuid.UUID(int=42))

def bridge():
    body = json.dumps({"user": USER, "endpoint": SERVER, "username": "axon", "password": "secret",
                       "apkg": base64.b64encode(open("/work/sample.apkg", "rb").read()).decode()}).encode()
    req = urllib.request.Request("http://axon-anki-bridge:8090/sync", body, {"Content-Type": "application/json"})
    return json.load(urllib.request.urlopen(req, timeout=120))

def anki_desktop(path, add_note=False):
    """A person's Anki desktop: download, optionally add a note, sync back."""
    col = Collection(path)
    auth = col.sync_login(username="axon", password="secret", endpoint=SERVER)
    out = col.sync_collection(auth, True)
    if out.required in (R.FULL_DOWNLOAD, R.FULL_SYNC):
        col.close_for_full_sync(); col.full_upload_or_download(auth=auth, server_usn=out.server_media_usn, upload=False); col.reopen(after_full_sync=True)
    if add_note:
        note = col.new_note(col.models.by_name("Basic"))
        note["Front"], note["Back"] = "Anki-only question", "kept"
        col.add_note(note, col.decks.id("Default"))
        col.sync_collection(auth, True)
    fronts = sorted(col.get_note(n).fields[0] for n in col.find_notes(""))
    media = col.media.dir()
    col.close()
    return fronts, media

first = bridge()
assert first["added"] == 2, first
fronts, _ = anki_desktop("/tmp/desktop1.anki2")
assert len(fronts) == 2 and any("Why space practice" in f for f in fronts), fronts

again = bridge()
assert again["added"] == 0 and again["updated"] + again["unchanged"] == 2, again

fronts, _ = anki_desktop("/tmp/desktop2.anki2", add_note=True)
assert len(fronts) == 3, fronts
bridge()
fronts, _ = anki_desktop("/tmp/desktop3.anki2")
assert len(fronts) == 3 and "Anki-only question" in fronts, fronts
print("Anki sync: cards delivered, updated without duplicates, Anki-only note kept")
PY
docker run --rm --network $net -v "$work:/work" axon-anki:check python /work/client.py
