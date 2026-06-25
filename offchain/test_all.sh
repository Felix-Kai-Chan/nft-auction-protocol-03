#!/bin/bash

echo "=== Health Check ==="
curl -s http://localhost:8080/health | jq '.'

echo -e "\n=== Auctions ==="
curl -s http://localhost:8080/api/auctions | jq '.data | length'

echo -e "\n=== Stats ==="
curl -s http://localhost:8080/api/bids/stats | jq '.'

echo -e "\n=== Database ==="
mysql -uroot -e "SELECT COUNT(*) as auctions FROM offchain.auctions;"
mysql -uroot -e "SELECT COUNT(*) as bids FROM offchain.bids;"