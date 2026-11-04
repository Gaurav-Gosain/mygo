#!/bin/sh
# fork-rename.sh: give this tree the fork's module path.
#
# The tuios branch of Gaurav-Gosain/mygo is an upstream release tag, the
# patch/* branches on top, and one last commit made by this script. To move
# to a new upstream tag, apply the patch branches on it and run this script
# again, then commit what it changed.
#
# It rewrites every Go import path, linker -X name and command that names the
# module (github.com/egoist/mygo) to github.com/Gaurav-Gosain/mygo, in every
# text file the repository tracks. URLs (https://github.com/egoist/mygo/...)
# stay as they are: the release assets they point at, such as libghostty-vt
# and SQLite, live upstream. npm package names (@egoist/mygo-*) stay too.
set -eu
cd "$(git rev-parse --show-toplevel)"
from=github.com/egoist/mygo
to=github.com/Gaurav-Gosain/mygo
git grep -l -I -z "$from" -- . ':!scripts/fork-rename.sh' |
	xargs -0 perl -pi -e "s{(?<!://)\\Q$from\\E(?![\\w-])}{$to}g"
# Imports sort by path, and the new path sorts elsewhere in its group.
git diff --name-only -z -- '*.go' | xargs -0 -r gofmt -w
echo "renamed: $(git diff --name-only | wc -l) files"
