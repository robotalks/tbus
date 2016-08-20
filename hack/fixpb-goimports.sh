#!/bin/bash

for f in go/tbus/*.pb.go; do
    sed -i '/^import _ "tbus\/common"$/d' $f
done
