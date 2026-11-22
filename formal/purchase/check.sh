#!/usr/bin/env bash
# Model-checks PurchaseSettlement.tla. Settle.cfg must hold; CheckThenAct.cfg
# must reproduce the double extension, which proves the invariant can fail.
set -euo pipefail

TLA_VERSION=1.7.4
TLA_SHA256=936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88

here="$(cd "$(dirname "$0")" && pwd)"
cache="${TLA_CACHE:-${TMPDIR:-/tmp}/tla2tools-$TLA_VERSION}"
jar="$cache/tla2tools.jar"
mkdir -p "$cache"
if [ ! -f "$jar" ]; then
  curl -fsSL -o "$jar.part" "https://github.com/tlaplus/tlaplus/releases/download/v$TLA_VERSION/tla2tools.jar"
  mv "$jar.part" "$jar"
fi
echo "$TLA_SHA256  $jar" | sha256sum -c --quiet -

run() {
  java -XX:+UseParallelGC -cp "$jar" tlc2.TLC -nowarning -workers 1 \
    -metadir "$(mktemp -d)" -config "$here/$1.cfg" "$here/PurchaseSettlement.tla"
}

if ! out="$(run Settle 2>&1)"; then
  echo "$out"
  echo "FAIL: Settle violates ExtendedAtMostOnce; a purchase can extend the subscription twice." >&2
  exit 1
fi
echo "ok: Settle holds ExtendedAtMostOnce"

if out="$(run CheckThenAct 2>&1)"; then
  echo "$out"
  echo "FAIL: the check-then-act model passed, so ExtendedAtMostOnce can no longer fail and proves nothing." >&2
  exit 1
fi
if ! grep -q "Invariant ExtendedAtMostOnce is violated" <<<"$out"; then
  echo "$out"
  echo "FAIL: CheckThenAct failed for a reason other than ExtendedAtMostOnce." >&2
  exit 1
fi
echo "ok: CheckThenAct reproduces the double extension"
