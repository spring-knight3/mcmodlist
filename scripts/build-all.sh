#!/bin/sh
# Build the Go engine and the Qt 6 GUI.
set -eu

cd "$(dirname "$0")/.."
mkdir -p build

go build -o build/mcmodlist-engine ./cmd/mcmodlist-engine

cmake -S qtgui -B build/qtgui -DCMAKE_BUILD_TYPE=Release
cmake --build build/qtgui -j

cat <<EOF
Built:
  build/mcmodlist-engine
  build/qtgui/mcmodlist-gui
Run tests with:
  QT_QPA_PLATFORM=offscreen MCMODLIST_ENGINE=$PWD/build/mcmodlist-engine ctest --test-dir build/qtgui --output-on-failure
EOF
