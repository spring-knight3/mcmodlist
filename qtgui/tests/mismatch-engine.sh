#!/bin/sh
# Stub engine that reports a wrong protocol version in hello.
while IFS= read -r line; do
  id=$(printf '%s' "$line" | sed -n 's/.*"id":\([0-9][0-9]*\).*/\1/p')
  printf '{"v":1,"id":%s,"result":{"protocol":99,"engineVersion":"stub","methods":[]}}\n' "$id"
done
