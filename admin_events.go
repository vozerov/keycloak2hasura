package main

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/machinebox/graphql"
	amqp "github.com/rabbitmq/amqp091-go"
	log "github.com/sirupsen/logrus"
)

type AdminEvent struct {
	Class                string          `json:"@class"`
	Timestamp            int64           `json:"time"`
	RealmID              string          `json:"realmId"`
	AuthDetails          json.RawMessage `json:"authDetails"`
	ResourceType         string          `json:"resourceType"`
	OperationType        string          `json:"operationType"`
	ResourcePath         string          `json:"resourcePath"`
	ResourceTypeAsString string          `json:"resourceTypeAsString"`
}

type AuthDetails struct {
	RealmID   string `json:"realmId"`
	ClientID  string `json:"clientId"`
	UserID    string `json:"userId"`
	IpAddress string `json:"ipAddress"`
}

func (eh *adminEventHandler) Handle(ctx context.Context, msg amqp.Delivery) interface{} {
	ev := AdminEvent{}

	err := json.Unmarshal(msg.Body, &ev)
	if err != nil {
		log.Errorf("Error on parsing json from queue: %s\n", err)
		msg.Nack(false, true)
	}

	ts := time.Unix(int64(ev.Timestamp/1000), 0)
	log.Infof("ADMIN EVENT: %s, TYPE: %s, REALMID: %s", ts.Format(time.RFC3339), ev.OperationType, ev.RealmID)

	switch ev.OperationType {
	case "DELETE":
		ad := AuthDetails{}
		err := json.Unmarshal(ev.AuthDetails, &ad)
		if err != nil {
			log.Errorf("Error on parsing json from queue: %s\n", err)
			msg.Nack(false, true)
		}

		userId := strings.Split(ev.ResourcePath, "/")[1]

		log.Infof("Removing user from hasura, userId: %s", userId)

		set := struct {
			Deleted bool `json:"deleted"`
		}{
			Deleted: true,
		}

		user := struct {
			ID string `json:"id"`
		}{
			ID: userId,
		}

		// make a request
		req := graphql.NewRequest(`
			mutation($set: user_set_input!, $user_id: user_pk_columns_input!){
				update_user_by_pk(
					_set: $set,
					pk_columns: $user_id
				) {
					id
				}
			}
			`)

		req.Var("set", set)
		req.Var("user_id", user)
		req.Header.Set("x-hasura-admin-secret", configuration.HasuraAdminSecret)
		ctx := context.Background()

		var response interface{}

		err = hc.Run(ctx, req, &response)
		if err != nil {
			log.Errorf("Update user error: %s\n", err)
			msg.Nack(false, true)
			return nil
		}

		log.Infof("Removed user")
	default:
		log.Infof("Do nothing")
	}

	// process message
	msg.Ack(false)

	return nil
}
