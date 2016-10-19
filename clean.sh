#!/usr/bin/env bash

set -e

# Cleanup local cluster state from data dirs
sudo rm -rf ./cluster/config/*
sudo rm -rf ./cluster/data/*
sudo rm -rf ./cluster/volumes/*
rm -f pvs.json
