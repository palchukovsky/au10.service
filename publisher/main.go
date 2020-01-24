package main

import (
	"flag"
	"strings"

	"bitbucket.org/au10/service/au10"
	"bitbucket.org/au10/service/postdb"
)

var (
	nodeName      = flag.String("name", "", "node instance name")
	streamBrokers = flag.String("stream_brokers",
		"localhost:9092", "stream brokers, as a comma-separated list")
	dbHost     = flag.String("db_host", "localhost", "database host")
	dbName     = flag.String("db_name", "postdb", "database name")
	dbLogin    = flag.String("db_login", "postdb", "database user login name")
	dbPassword = flag.String("db_password", "", "database user login password")
)

const nodeType = "publisher"

func main() {
	flag.Parse()

	service := au10.DialOrPanic(nodeType, *nodeName,
		strings.Split(*streamBrokers, ","), au10.NewFactory())
	defer service.Close()

	service.Log().Debug(`Connecting to the database "%s@%s/%s"...`,
		*dbLogin, *dbHost, *dbName)
	db, err := postdb.Dial(*dbHost, *dbName, *dbLogin, *dbPassword)
	if err != nil {
		service.Log().Fatal(`Failed to connect to the database "%s@%s/%s": "%s".`,
			*dbLogin, *dbHost, *dbName, err)
		return
	}
	defer db.Close()

	var stream au10.PublishStream
	stream, err = au10.NewPublishStreamSingleton(service)
	if err != nil || stream == nil {
		service.Log().Fatal(`Failed to start publish stream: "%s".`, err)
		return
	}
	defer stream.Close()

	var notifier au10.PostNotifier
	notifier, err = au10.NewPostNotifier(service)
	if err != nil {
		service.Log().Fatal(`Failed to start notifier: "%s".`, err)
		return
	}

	service.Log().Debug("Starting...")
	defer service.Log().Info("Stopping...")
	for {
		select {
		case post, isOpen := <-stream.GetRecordsChan():
			if !isOpen {
				return
			}
			if err := storePost(post, db); err != nil {
				service.Log().Error(`Failed to store post: "%s".`, err)
			} else if len(post.GetMessages()) == 0 {
				if err := notifier.PushVocal(post); err != nil {
					service.Log().Error(`Failed to notify %d post: "%s".`,
						post.GetID(), err)
				}
			}
		case err := <-stream.GetErrChan():
			service.Log().Error(`Processing stream error: "%s"...`, err)
			return
		}
	}
}

func storePost(post au10.Vocal, db postdb.DB) error {
	trans, err := db.Begin()
	if err != nil {
		return err
	}
	if err = trans.InsertPost(post); err != nil {
		return err
	}
	return trans.Commit()
}
