#!/bin/bash
set -e

API=http://localhost:8080

echo "== register =="
REG=$(curl -s -X POST $API/api/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","email":"alice@example.com","password":"secret123"}')
echo "$REG"
TOKEN=$(echo "$REG" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
[ -z "$TOKEN" ] && { echo "no token, abort"; exit 1; }

echo "== login =="
LOGIN=$(curl -s -X POST $API/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"secret123"}')
echo "$LOGIN"

AUTH="Authorization: Bearer $TOKEN"

echo "== list products =="
PRODUCTS=$(curl -s "$API/api/catalog/products?limit=3" -H "$AUTH")
echo "$PRODUCTS"
PID=$(echo "$PRODUCTS" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
[ -z "$PID" ] && { echo "no product, abort"; exit 1; }

echo "== add to cart =="
curl -s -X POST "$API/api/cart/items" -H "$AUTH" \
  -H 'Content-Type: application/json' \
  -d "{\"product_id\":\"$PID\",\"quantity\":2,\"price_cents\":12900}"
echo

echo "== view cart =="
curl -s "$API/api/cart" -H "$AUTH"
echo

echo "== create order (should succeed) =="
ORDER=$(curl -s -X POST "$API/api/orders" -H "$AUTH" \
  -H 'Content-Type: application/json' \
  -d "{\"items\":[{\"product_id\":\"$PID\",\"quantity\":2,\"price_cents\":12900}]}")
echo "$ORDER"
OID=$(echo "$ORDER" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

echo "== wait 2s for saga =="
sleep 2

echo "== order status =="
curl -s "$API/api/orders/$OID" -H "$AUTH"
echo

echo "smoke OK"
