#!/bin/sh
# Print the package directory of every _examples/ program, one per line,
# relative to the repository root and prefixed with "./" so `go vet` and
# `golangci-lint run` treat each one as a package directory.
#
# Go package patterns ignore directories whose name begins with "_", so
# neither `./...` nor `./_examples/...` matches these packages; the analyzers
# only see them when the directories are named explicitly. Enumerating them
# here — rather than listing them in .github/workflows/ci.yml and the
# Makefile — means a new _examples/<program>/ is analyzed the moment it is
# added, and keeps CI and `make lint` looking at exactly the same set.
#
# A directory counts as a program when it holds at least one .go file, so a
# fixture or data directory under _examples/ cannot break the gate. Exits 1
# when nothing matches, because an empty list would silently analyze nothing.
set -eu

root="$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$root"

dirs=""
for dir in _examples/*/; do
	dir="${dir%/}"
	[ -d "$dir" ] || continue
	set -- "$dir"/*.go
	[ -e "$1" ] || continue
	dirs="${dirs}./${dir}
"
done

if [ -z "$dirs" ]; then
	echo "example-dirs.sh: no Go program directories found under _examples/" >&2
	exit 1
fi

printf '%s' "$dirs"
