#!/bin/bash
# Fix linter issues in the code

# Fix SetReadDeadline unchecked errors in socks_test.go
sed -i.bak 's/err = \([a-zA-Z]*\)\.SetReadDeadline(\([^)]*\))/if err = \1.SetReadDeadline(\2); err != nil/g' /root/latency-space/proxy/src/socks_test.go
sed -i.bak 's/err = \([a-zA-Z]*\)\.SetReadDeadline(\([^)]*\))/if err = \1.SetReadDeadline(\2); err != nil/g' /root/latency-space/proxy/src/udp_failure_test.go
sed -i.bak 's/err = \([a-zA-Z]*\)\.SetReadDeadline(\([^)]*\))/if err = \1.SetReadDeadline(\2); err != nil/g' /root/latency-space/proxy/src/extended_socks_test.go
sed -i.bak 's/err = \([a-zA-Z]*\)\.SetReadDeadline(\([^)]*\))/if err = \1.SetReadDeadline(\2); err != nil/g' /root/latency-space/proxy/src/auth_latency_test.go

echo "Fixed SetReadDeadline issues"