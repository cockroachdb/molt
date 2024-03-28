#!/usr/bin/env bash
set -euxo pipefail
set -m

VARIANT=${1:-base_aws}

# Get this current directory.
DIR="$(cd "$(dirname "$0")" && pwd)"

# Add new case statements for new variants
case $VARIANT in
  base_aws)
    $DIR/setup-scale-test.sh --cluster-name migrations-nightly-base-aws -n 4 --cloud aws --zones us-east-1a
    ;;

  scaled_aws)
    $DIR/setup-scale-test.sh --cluster-name migrations-nightly-scaled-aws -n 4 --cloud aws --volume-size 1500 --zones us-east-1a --aws-cpu-options 'CoreCount=8,ThreadsPerCore=2' --machine-type m6idn.4xlarge --aws-machine-type-ssd m6idn.4xlarge
    ;;

  *)
    echo -n "unknown test variant"
    exit 1
    ;;
esac

exit 0