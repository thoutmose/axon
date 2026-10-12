"""Axon's Anki bridge (EP-42).

A small HTTP service, reachable only on the Compose network, that keeps one
Anki collection per person and syncs it with their anki-sync-server account
using the official Anki library. Axon sends an .apkg of the person's cards;
the bridge imports it, updating notes in place by their stable GUIDs and never
deleting cards that exist only in Anki, then syncs.

POST /sync  {"user": uuid, "endpoint": url, "username": str, "password": str, "apkg": base64}
GET  /healthz
"""

import base64
import json
import os
import re
import tempfile
import threading
import uuid
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

from anki.collection import Collection
from anki.import_export_pb2 import (
    ImportAnkiPackageOptions,
    ImportAnkiPackageRequest,
    ImportAnkiPackageUpdateCondition,
)
from anki.sync_pb2 import SyncCollectionResponse

DATA = os.environ.get("ANKI_BRIDGE_DATA", "/data")
MAX_BODY = 64 << 20
locks: dict[str, threading.Lock] = {}
locks_guard = threading.Lock()
Required = SyncCollectionResponse.ChangesRequired


def lock_for(user: str) -> threading.Lock:
    with locks_guard:
        return locks.setdefault(user, threading.Lock())


def full_sync(col: Collection, auth, out, upload: bool) -> None:
    col.close_for_full_sync()
    col.full_upload_or_download(auth=auth, server_usn=out.server_media_usn, upload=upload)
    col.reopen(after_full_sync=True)


def sync(col: Collection, auth, fresh: bool) -> str:
    """One sync, resolving full-sync requests safely.

    A new local collection always takes the server's copy (download); an upload
    only happens when the server has nothing, so the person's Anki is never
    overwritten.
    """
    out = col.sync_collection(auth, True)
    if out.required == Required.FULL_DOWNLOAD or (out.required == Required.FULL_SYNC and fresh):
        full_sync(col, auth, out, upload=False)
        return "downloaded"
    if out.required == Required.FULL_UPLOAD:
        full_sync(col, auth, out, upload=True)
        return "uploaded"
    if out.required == Required.FULL_SYNC:
        raise RuntimeError("Anki asks for a full sync; open Anki desktop, sync once, then try again")
    return "synced"


def run(req: dict) -> dict:
    user = str(uuid.UUID(req["user"]))  # rejects anything that is not a UUID
    endpoint, username, password = req["endpoint"], req["username"], req["password"]
    if not re.match(r"^https?://", endpoint):
        raise ValueError("the sync server address must start with http:// or https://")
    apkg = base64.b64decode(req["apkg"], validate=True)
    path = os.path.join(DATA, f"{user}.anki2")
    with lock_for(user):
        fresh = not os.path.exists(path)
        col = Collection(path)
        try:
            auth = col.sync_login(username=username, password=password, endpoint=endpoint)
            first = sync(col, auth, fresh)
            with tempfile.NamedTemporaryFile(suffix=".apkg", dir=DATA) as f:
                f.write(apkg)
                f.flush()
                result = col.import_anki_package(
                    ImportAnkiPackageRequest(
                        package_path=f.name,
                        options=ImportAnkiPackageOptions(
                            with_scheduling=False,
                            merge_notetypes=True,
                            update_notes=ImportAnkiPackageUpdateCondition.IMPORT_ANKI_PACKAGE_UPDATE_CONDITION_ALWAYS,
                            update_notetypes=ImportAnkiPackageUpdateCondition.IMPORT_ANKI_PACKAGE_UPDATE_CONDITION_ALWAYS,
                        ),
                    )
                )
            second = sync(col, auth, False)
            return {
                "added": len(result.log.new),
                "updated": len(result.log.updated),
                "unchanged": len(result.log.duplicate),
                "steps": [first, second],
            }
        finally:
            col.close()


class Handler(BaseHTTPRequestHandler):
    def reply(self, code: int, body: dict) -> None:
        data = json.dumps(body).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def do_GET(self) -> None:
        if self.path == "/healthz":
            self.reply(200, {"status": "ok"})
        else:
            self.reply(404, {"error": "not found"})

    def do_POST(self) -> None:
        if self.path != "/sync":
            self.reply(404, {"error": "not found"})
            return
        length = int(self.headers.get("Content-Length") or 0)
        if length <= 0 or length > MAX_BODY:
            self.reply(413, {"error": "the package is too large"})
            return
        try:
            self.reply(200, run(json.loads(self.rfile.read(length))))
        except (KeyError, ValueError) as exc:
            self.reply(400, {"error": str(exc)})
        except Exception as exc:  # the sync server's message is safe to show
            self.reply(502, {"error": str(exc)})

    def log_message(self, fmt: str, *args) -> None:  # no request bodies or credentials in logs
        print(f"anki-bridge {self.command} {self.path} -> {args[1] if len(args) > 1 else ''}", flush=True)


if __name__ == "__main__":
    os.makedirs(DATA, exist_ok=True)
    ThreadingHTTPServer(("0.0.0.0", int(os.environ.get("PORT", "8090"))), Handler).serve_forever()
