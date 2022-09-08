package main

import (
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

type Configuration struct {
	LogLevel                  string `envconfig:"LOG_LEVEL" default:"info"`
	RabbitMQURI               string `envconfig:"RABBITMQ_URI" default:"amqp://user:pass@127.0.0.1:5672/"`
	RabbitMQClientEventsQueue string `envconfig:"RABBITMQ_CLIENT_EVENTS_QUEUE" default:"keycloak-client-events"`
	RabbitMQAdminEventsQueue  string `envconfig:"RABBITMQ_ADMIN_EVENTS_QUEUE" default:"keycloak-admin-events"`
	HasuraGraphQLEndpoint     string `envconfig:"HASURA_GRAPHQL_ENDPOINT" defult:"http://127.0.0.1:8080/v1/graphql"`
	HasuraAdminSecret         string `envconfig:"HASURA_ADMIN_SECRET" default:""`
}

func (c *Configuration) Load() {
	err := envconfig.Process("", c)
	if err != nil {
		log.Fatalf("Can't parse configuration: %s", err)
	}
}
