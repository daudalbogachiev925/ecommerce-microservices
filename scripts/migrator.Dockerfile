FROM postgres:16-alpine

WORKDIR /migrations

# копируем все .sql из всех сервисов, сохраняя путь
COPY services/users/migrations /migrations/users
COPY services/catalog/migrations /migrations/catalog
COPY services/cart/migrations /migrations/cart
COPY services/orders/migrations /migrations/orders
COPY services/payments/migrations /migrations/payments
COPY services/notifications/migrations /migrations/notifications

COPY scripts/run-migrations.sh /run-migrations.sh
RUN chmod +x /run-migrations.sh

ENTRYPOINT ["/run-migrations.sh"]
