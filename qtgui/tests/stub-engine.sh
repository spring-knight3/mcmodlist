#!/bin/sh
# Stub mcmodlist-engine for Qt client tests. Speaks JSON-Lines on stdin/stdout.
while IFS= read -r line; do
  id=$(printf '%s' "$line" | sed -n 's/.*"id":\([0-9][0-9]*\).*/\1/p')
  case "$line" in
    *'"method":"hello"'*)
      printf '{"v":1,"id":%s,"result":{"protocol":1,"engineVersion":"stub","methods":["hello","slow","split","die","big"]}}\n' "$id" ;;
    *'"method":"slow"'*)
      sleep 1
      printf '{"v":1,"id":%s,"result":{"done":true}}\n' "$id" ;;
    *'"method":"split"'*)
      printf '{"v":1,"id":%s,"res' "$id"
      sleep 0.3
      printf 'ult":{"split":true}}\n' "$id" ;;
    *'"method":"view"'*)
      sleep 0.4
      printf '{"v":1,"id":%s,"result":{"state":{"marker":%s,"rows":[],"groups":[],"loaders":[],"loaderOptions":[],"total":0,"instance":{"name":"stub"}}}}\n' "$id" "$id" ;;
    *'"method":"die"'*)
      printf '{"v":1,"id":%s,"result":{"bye":true}}\n' "$id"
      exit 3 ;;
    *'"method":"big"'*)
      printf '{"v":1,"id":%s,"result":{"blob":"' "$id"
      head -c 300000 /dev/zero | tr '\0' 'x'
      printf '"}}\n' "$id" ;;
    *)
      printf '{"v":1,"id":%s,"error":{"code":"invalid_request","message":"unknown method"}}\n' "$id" ;;
  esac
done
