#!/usr/bin/env bash
# query_facts.sh — Query knowledge facts from a Zerone node via REST API.
#
# Usage:
#   ./query_facts.sh                      # list all facts
#   ./query_facts.sh fact-001             # get a single fact by ID
#
# Requires: curl, jq (optional, for pretty-printing)

NODE="${ZERONE_NODE:-http://localhost:1317}"

if [ -n "$1" ]; then
  echo "==> Querying fact: $1"
  curl -s "$NODE/zerone/knowledge/v1/facts/$1" | jq .
else
  echo "==> Listing all facts"
  curl -s "$NODE/zerone/knowledge/v1/facts" | jq .

  echo ""
  echo "==> Listing all domains"
  curl -s "$NODE/zerone/knowledge/v1/domains" | jq .

  echo ""
  echo "==> Module parameters"
  curl -s "$NODE/zerone/knowledge/v1/params" | jq .
fi
