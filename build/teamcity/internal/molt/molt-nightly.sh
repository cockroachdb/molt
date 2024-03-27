#!/usr/bin/env bash
set -euxo pipefail
set -m

# Get this current directory.
DIR="$(cd "$(dirname "$0")" && pwd)"

# Run the setup script.
$DIR/setup-scale-test.sh -n 4 -z us-east-1a