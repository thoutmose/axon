#!/bin/sh
# The French README must keep the English one's structure: the same heading
# levels in the same order, and the same code blocks and tables.
set -eu
shape() {
	awk '
		/^```/ { fence = !fence; if (fence) print "code"; next }
		fence { next }
		/^#+ / { match($0, /^#+/); print "h" RLENGTH; next }
		/^\|---/ { print "table" }
	' "$1"
}
shape README.md >/tmp/readme.en
shape README.fr.md >/tmp/readme.fr
if ! cmp -s /tmp/readme.en /tmp/readme.fr; then
	echo "README.fr.md is out of step with README.md (headings, code blocks, or tables differ):" >&2
	diff /tmp/readme.en /tmp/readme.fr >&2 || true
	exit 1
fi
grep -q 'href="README.fr.md"' README.md || { echo "README.md must link to README.fr.md" >&2; exit 1; }
grep -q 'href="README.md"' README.fr.md || { echo "README.fr.md must link to README.md" >&2; exit 1; }
