package main

import (
	"fmt"
	"os"
	"os/signal"
	"path"
	"runtime"
	"strings"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/machinebox/graphql"
	"github.com/makasim/amqpextra"
	"github.com/makasim/amqpextra/consumer"
	log "github.com/sirupsen/logrus"
	"github.com/zput/zxcTool/ztLog/zt_formatter"
)

// struct for handling events
type clientEventHandler struct{}
type adminEventHandler struct{}

var (
	hc            *graphql.Client
	configuration Configuration
)

func init() {
	var formatter log.Formatter

	formatter = &zt_formatter.ZtFormatter{
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			filename := path.Base(f.File)
			fs := strings.Split(fmt.Sprintf(f.Function), "/")
			return fmt.Sprintf("%-45s", fmt.Sprintf("%s()", fs[len(fs)-1])), fmt.Sprintf("%15s:%03d", filename, f.Line)
		},
		Formatter: nested.Formatter{
			//HideKeys: true,
			FieldsOrder: []string{"component", "category"},
		},
	}

	log.SetFormatter(formatter)
	log.SetReportCaller(true)

	log.SetOutput(os.Stdout)

	lvl, err := log.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		lvl = log.DebugLevel
	}
	log.SetLevel(lvl)
}

func main() {
	log.Infof("Loading configuration")
	configuration.Load()

	log.Info("Connecting to rabbitmq...")
	dialer, err := amqpextra.NewDialer(amqpextra.WithURL(configuration.RabbitMQURI))
	if err != nil {
		log.Fatalf("Can't create dialer from amqpextra")
	}
	defer dialer.Close()
	log.Infof("connected to rabbitmq")

	log.Infof("Creating hasura graphql client")
	hc = graphql.NewClient(configuration.HasuraGraphQLEndpoint)
	hc.Log = func(s string) { log.Infof(s) }
	log.Infof("Created hasura graphql client")

	// creating consumer for clients queue
	ehc := clientEventHandler{}
	cc, err := dialer.Consumer(
		consumer.WithQueue(configuration.RabbitMQClientEventsQueue),
		consumer.WithHandler(&ehc),
	)
	defer cc.Close()

	// creating consumer for admin queue
	eha := adminEventHandler{}
	ca, err := dialer.Consumer(
		consumer.WithQueue(configuration.RabbitMQAdminEventsQueue),
		consumer.WithHandler(&eha),
	)
	defer ca.Close()

	if err != nil {
		fmt.Printf("Error on consuming messages: %s\n", err)
	}

	// trap SIGINT to trigger a shutdown.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, os.Kill)

	// make rabbitmq events
	dEvents := dialer.Notify(make(chan amqpextra.State, 1))

	log.Printf("Start consuming messages")
	for {
		select {
		case <-signals:
			log.Printf("Got system signal, exiting...\n")
			return
		case state := <-dEvents:
			if state.Ready != nil {
				log.Info("Dialer is ready")
			}
			if state.Unready != nil {
				log.Errorf("Dialer is not ready: %s\n", state.Unready.Err)
			}
		}

	}
}
