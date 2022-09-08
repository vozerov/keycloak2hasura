package main

import (
	"context"
	"encoding/json"
	"time"

	"github.com/machinebox/graphql"
	amqp "github.com/rabbitmq/amqp091-go"
	log "github.com/sirupsen/logrus"
)

type ClientEvent struct {
	Class     string          `json:"@class"`
	Timestamp int64           `json:"time"`
	Type      string          `json:"type"`
	RealmID   string          `json:"realmId"`
	ClientID  string          `json:"clientId"`
	UserID    string          `json:"userId"`
	SessionID string          `json:"sessionId"`
	IpAddress string          `json:"ipAddress"`
	Details   json.RawMessage `json:"details"`
}

type LoginDetails struct {
	AuthMethod   string `json:"auth_method"`
	AuthType     string `json:"auth_type"`
	ResponseType string `json:"code"`
	RedirectUri  string `json:"redirect_uri"`
	Consent      string `json:"consent"`
	CodeID       string `json:"code_id"`
	Username     string `json:"username"`
	ResponseMode string `json:"response_mode"`
}

type RegisterDetails struct {
	AuthMethod     string `json:"auth_method"`
	AuthType       string `json:"auth_type"`
	RegisterMethod string `json:"register_method"`
	LastName       string `json:"last_name"`
	RedirectUri    string `json:"redirect_uri"`
	FirstName      string `json:"first_name"`
	CodeID         string `json:"code_id"`
	Email          string `json:"email"`
	Username       string `json:"username"`
}

type VerifyEmailDetails struct {
	AuthMethod   string `json:"auth_method"`
	TokenID      string `json:"token_id"`
	Action       string `json:"action"`
	ResponseType string `json:"response_type"`
	RedirectUri  string `json:"redirect_uri"`
	RememberMe   string `json:"remember_me"`
	Consent      string `json:"consent"`
	CodeID       string `json:"code_id"`
	Email        string `json:"email"`
	ResponseMode string `json:"response_mode"`
	Username     string `json:"username"`
}

func (eh *clientEventHandler) Handle(ctx context.Context, msg amqp.Delivery) interface{} {
	ev := ClientEvent{}

	err := json.Unmarshal(msg.Body, &ev)
	if err != nil {
		log.Errorf("Error on parsing json from queue: %s\n", err)
		msg.Nack(false, true)
	}

	ts := time.Unix(int64(ev.Timestamp/1000), 0)
	log.Infof("CLIENT EVENT: %s, TYPE: %s, USERID: %s", ts.Format(time.RFC3339), ev.Type, ev.UserID)

	switch ev.Type {
	case "LOGIN":
		ld := LoginDetails{}
		err := json.Unmarshal(ev.Details, &ld)
		if err != nil {
			log.Errorf("Error on parsing json from queue: %s\n", err)
			msg.Nack(false, true)
		}

		log.Infof("Login in keycloak, do nothing")
	case "REGISTER":
		rd := RegisterDetails{}
		err := json.Unmarshal(ev.Details, &rd)
		if err != nil {
			log.Errorf("Error on parsing json from queue: %s\n", err)
			msg.Nack(false, true)
		}

		log.Infof("Creating Creating user in Hasura: ID: %s, firstName: %s, lastName: %s, email: %s", ev.UserID, rd.FirstName, rd.LastName, rd.Email)

		user := struct {
			ID            string `json:"id"`
			FirstName     string `json:"first_name"`
			LastName      string `json:"last_name"`
			Username      string `json:"username"`
			Email         string `json:"email"`
			EmailVerified bool   `json:"email_verified"`
			Deleted       bool   `json:"deleted"`
		}{
			ID:            ev.UserID,
			FirstName:     rd.FirstName,
			LastName:      rd.LastName,
			Username:      rd.Username,
			Email:         rd.Email,
			EmailVerified: false,
			Deleted:       false,
		}

		// make a request
		req := graphql.NewRequest(`
			mutation($user: user_insert_input!){
				insert_user(
					objects: [$user],
					on_conflict: {
						constraint: user_pkey,
						update_columns: [first_name, last_name]
					  }				  
			) {
			  		returning {
						id
			  		}
				}
		  	}
		`)

		req.Var("user", user)
		req.Header.Set("x-hasura-admin-secret", configuration.HasuraAdminSecret)
		ctx := context.Background()

		response := struct {
			InsertUser struct {
				Returning []struct {
					ID string `json:"id"`
				} `json:"returning"`
			} `json:"insert_user"`
		}{}

		err = hc.Run(ctx, req, &response)
		if err != nil {
			log.Errorf("Create user error: %s\n", err)
			msg.Nack(false, true)
			return nil
		}

		log.Info("Successfully created user: %+v\n", response)
	case "VERIFY_EMAIL":
		ved := VerifyEmailDetails{}
		err := json.Unmarshal(ev.Details, &ved)
		if err != nil {
			log.Errorf("Error on parsing json from queue: %s\n", err)
			msg.Nack(false, true)
		}

		log.Infof("Updating user with email verified: ID: %s", ev.UserID)

		set := struct {
			EmailVerified bool `json:"email_verified"`
		}{
			EmailVerified: true,
		}

		user := struct {
			ID string `json:"id"`
		}{
			ID: ev.UserID,
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

		log.Infof("Updating user with email verified")
	default:
		log.Infof("Do nothing")
	}

	// process message
	msg.Ack(false)

	return nil
}
