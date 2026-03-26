#!/usr/bin/env bash
set -euo pipefail

DEST="./temp.zip"

echo "Starting downloads into $DEST ..."

wget https://cloud.ilabt.imec.be/index.php/s/qm8EGWCZBot9Hjj/download -O $DEST

echo "All downloads complete."

echo "Unzip the archive ."

unzip ./temp.zip -d ./

rm ./temp.zip

echo "Unzip the archive."

echo "Setting the repository architecture."
mv ./large-rdf-bench/queries queries
mv ./large-rdf-bench/sources sources
rm -r ./large-rdf-bench
