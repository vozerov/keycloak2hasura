General
=======

Keycloak2Hasura app allows you to sync users from keycloak instance to hasura table through graphql queries. So the process would be:

1. User goes to keycloak auth page and creates an account
2. Keycloak rabbitmq plugin will send event about registration to rabbitmq
3. keycloak2hasura consumes this messages and runs graphql queries against hasura instance to create users

Keycloak
========

You have to build custom keycloak image with rabbitmq plugin installed. It will publish all messages from keycloak directly to rabbitmq.

Dockerfile:
```
FROM quay.io/keycloak/keycloak:latest as builder

ENV KC_HEALTH_ENABLED=true
ENV KC_METRICS_ENABLED=true
ENV KC_FEATURES=token-exchange
ENV KC_DB=postgres

# Install custom providers
RUN curl -sL https://github.com/aerogear/keycloak-metrics-spi/releases/download/2.5.3/keycloak-metrics-spi-2.5.3.jar -o /opt/keycloak/providers/keycloak-metrics-spi-2.5.3.jar
RUN curl -sL https://github.com/aznamier/keycloak-event-listener-rabbitmq/releases/download/3.0/keycloak-to-rabbit-3.0.jar -o /opt/keycloak/providers/keycloak-to-rabbit-3.0.jar

RUN /opt/keycloak/bin/kc.sh build

FROM quay.io/keycloak/keycloak:latest
COPY --from=builder /opt/keycloak/ /opt/keycloak/


WORKDIR /opt/keycloak

ENTRYPOINT ["/opt/keycloak/bin/kc.sh"]
```

After build you have to configure this plugin. Follow this steps https://github.com/aznamier/keycloak-event-listener-rabbitmq.

Rabbitmq
========

In rabbitmq you can use `amq.topic` exchange to publish messages from keycloak. Just create 2 classic queues for `client` and `admin` events from your realm.

Bind `admin` queue to `amq.topic` with `KK.EVENT.ADMIN.#` routing key.
Bind `client` queue to `amq.topic` with `KK.EVENT.CLIENT.#` routing key.

More info you can find at https://github.com/aznamier/keycloak-event-listener-rabbitmq.

Hasura
======

Hasura setup is pretty simple, just don't forget to add admin secret env - it will be used for authentication.

Sample docker-compose.yml for hasura should look like:

```
version: '3.6'
services:
  graphql-engine:
    image: hasura/graphql-engine:v2.10.1
    restart: always
    environment:
      HASURA_GRAPHQL_METADATA_DATABASE_URL: postgres://user:pass@postgres/db?sslmode=disable
      HASURA_GRAPHQL_ENABLE_CONSOLE: "true"
      HASURA_GRAPHQL_ADMIN_SECRET: SECRET_KEY
      HASURA_GRAPHQL_JWT_SECRET: '{"type":"RS256", "jwk_url":"https://keycloak.local/realms/hasura/protocol/openid-connect/certs"}'
```

Keycloak2Hasura
===============

k2h supports the following environment variables:

- `LOG_LEVEL` - log level, `info` by default
- `RABBITMQ_URI` - rabbitmq uri, default: `amqp://user:pass@127.0.0.1:5672/"`
- `RABBITMQ_CLIENT_EVENTS_QUEUE` - queue for client events, default: `keycloak-client-events`
- `RABBITMQ_ADMIN_EVENTS_QUEUE` - queue for admin events, default: `keycloak-admin-events`
- `HASURA_GRAPHQL_ENDPOINT` - endpoint for hasura's graphql, default: `http://127.0.0.1:8080/v1/graphql`
- `HASURA_ADMIN_SECRET` - hasura's admin secret key which was set during hasura startup.

k2h also has dockerfile which allows you to build your own docker container and run it with env vars mentioned above.