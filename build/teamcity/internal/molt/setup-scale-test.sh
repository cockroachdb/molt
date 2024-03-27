#!/usr/bin/env bash
set -euxo pipefail
set -m

export ROACHPROD_USER=migrations
# Read flags
CLUSTER=migrations-nightly-final
PROVIDER="aws"
ZONE="us-east-1a"
NUM_NODES=4

# TODO instance size and settings.
while getopts c:p:z:n: flag
do
    case "${flag}" in
        c) CLUSTER=${OPTARG};;
        p) PROVIDER=${OPTARG};;
        z) ZONE=${OPTARG};;
        n) NUM_NODES=${OPTARG};;
    esac
done

export CLUSTER

echo "Listing out all parameters"
echo "Cloud Provider: $PROVIDER";
echo "Zone: $ZONE";
echo "Num Nodes: $NUM_NODES";
echo "Cluster: $CLUSTER";

# Setup Auth
echo "$GOOGLE_EPHEMERAL_CREDENTIALS" > creds.json
gcloud auth activate-service-account --key-file=creds.json

aws configure set aws_access_key_id %env.AWS_ACCESS_KEY_ID%;
aws configure set aws_secret_access_key %env.AWS_SECRET_ACCESS_KEY%;
aws configure set default.region "US-EAST-1";
mkdir -p ~/.ssh/

ssh-keygen -t rsa -q -f "$HOME/.ssh/id_rsa" -N ""
ls ~/.ssh/
# Download Roachprod Binary
gcloud storage cp gs://migrations-fetch-ci-test/roachprod-binary/roachprod roachprod

# TODO: Build the binary for MOLT
# go build -o ./molt .

# Download the latest MOLT image for now.
wget https://molt.cockroachdb.com/molt/cli/molt-latest.linux-amd64.tgz
tar -xvzf "molt-latest.linux-amd64.tgz"

# We want to clean up the cluster and roachprod binary.
trap "./roachprod destroy $CLUSTER && rm roachprod" EXIT

# Setup Roachprod
chmod +x roachprod

# Conditional logic for AWS vs. GCE.
if [[ "$PROVIDER" == "aws" ]]; then
    ./roachprod create $CLUSTER --clouds $PROVIDER --aws-zones $ZONE -n $NUM_NODES
elif [[ "$PROVIDER" == "gcp" ]]; then
    ./roachprod create $CLUSTER --clouds $PROVIDER --gce-zones $ZONE -n $NUM_NODES
fi

./roachprod stage $CLUSTER release v23.2.2 --os linux
./roachprod start $CLUSTER

# Put binary on
./roachprod put $CLUSTER molt

exit 0