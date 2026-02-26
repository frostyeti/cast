#!/bin/bash
cat << 'INNEREOF' > patch.go
package main
// just a dummy script to remove the separate flag definitions and add persistent ones
INNEREOF
