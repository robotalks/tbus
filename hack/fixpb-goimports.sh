#!/bin/bash

for f in go/tbus/*.pb.go; do
    sed -i '/^import _ "common"$/d' $f
done
