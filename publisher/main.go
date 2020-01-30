package main

import (
	"errors"
	"flag"
	"strings"
	"sync"

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

func main() {
	flag.Parse()

	service := au10.DialOrPanic("publisher", *nodeName,
		strings.Split(*streamBrokers, ","), au10.NewFactory())
	defer service.Close()
	defer service.Log().Info("Stopped.")

	service.Log().Debug(`Connecting to the database "%s@%s/%s"...`,
		*dbLogin, *dbHost, *dbName)
	db, err := postdb.Dial(*dbHost, *dbName, *dbLogin, *dbPassword,
		func(err error) { service.Log().Fatal(`DB error: "%s".`, err) })
	if err != nil {
		service.Log().Fatal(`Failed to connect to the database "%s@%s/%s": "%s".`,
			*dbLogin, *dbHost, *dbName, err)
		return
	}
	defer db.Close()

	users, err := au10.NewUsers(service)
	if err != nil {
		service.Log().Fatal(`Failed to start users service: "%s".`, err)
		return
	}
	defer users.Close()

	posts := au10.NewPosts(service, db)
	if err != nil {
		service.Log().Fatal(`Failed to start posts service: "%s".`,
			*dbLogin, *dbHost, *dbName, err)
		return
	}
	defer posts.Close()

	var postsStream au10.PublishPostsStream
	postsStream, err = au10.NewPublishPostsStreamSingleton(service)
	if err != nil || postsStream == nil {
		service.Log().Fatal(`Failed to start posts stream: "%s".`, err)
		return
	}
	defer postsStream.Close()

	var messagesStream au10.PublishMessagesStream
	messagesStream, err = au10.NewPublishMessagesStreamSingleton(service)
	if err != nil || messagesStream == nil {
		service.Log().Fatal(`Failed to start messages stream: "%s".`, err)
		return
	}
	defer messagesStream.Close()

	var notifier au10.PostNotifier
	notifier, err = au10.NewPostNotifier(service)
	if err != nil {
		service.Log().Fatal(`Failed to start notifier: "%s".`, err)
		return
	}

	stopBarrier := sync.WaitGroup{}
	defer service.Log().Debug("Stopping...")
	defer stopBarrier.Wait()

	service.Log().Debug("Started.")
	for {
		select {

		case post, isOpen := <-postsStream.GetRecordsChan():
			if !isOpen {
				return
			}
			if err := storePost(post, db); err != nil {
				service.Log().Error(`Failed to store post %d: "%s".`, post.GetID(), err)
			} else {
				service.Log().Debug(`Stored post %d.`, post.GetID())
				if len(post.GetMessages()) == 0 {
					err := notifier.PushVocal(post)
					if err != nil {
						service.Log().Error(`Failed to publish post %d: "%s".`,
							post.GetID(), err)
					} else {
						service.Log().Debug(`Published post %d.`, post.GetID())
					}
				}
			}

		case message, isOpen := <-messagesStream.GetRecordsChan():
			if !isOpen {
				return
			}
			user, err := checkMessage(message, users, posts)
			if err != nil {
				service.Log().Error(
					`Failed to store %d bytes chunk for message %d/%d from %d: "%s".`,
					len(message.Data), message.Post, message.ID, message.Author, err)
				if user != nil {
					user.BlockByProtocolMismatch()
				}
			} else if err := storeMessage(message, db); err != nil {
				service.Log().Error(
					`Failed to store %d bytes chunk for message %d/%d from %d: "%s".`,
					len(message.Data), message.Post, message.ID, message.Author, err)
			} else {
				service.Log().Debug(`Stored %d bytes chunk for message %d/%d from %d.`,
					len(message.Data), message.Post, message.ID, message.Author)
			}

		case err := <-postsStream.GetErrChan():
			service.Log().Error(`Posts processing stream error: "%s"...`, err)
			return
		case err := <-messagesStream.GetErrChan():
			service.Log().Error(`Messages chunks processing stream error: "%s".`, err)
			return
		}
	}
}

func storePost(post au10.Vocal, db postdb.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	location := post.GetLocation()
	err = tx.InsertPost(
		uint32(post.GetID()), uint32(post.GetAuthor()),
		location.Latitude, location.Longitude)
	if err != nil {
		return err
	}
	for _, m := range post.GetMessages() {
		err = tx.InsertMessage(
			uint32(m.GetID()), uint32(post.GetID()), uint32(m.GetKind()), m.GetSize())
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func checkMessage(
	message *au10.PublisherMessageData,
	users au10.Users,
	posts au10.Posts) (au10.User, error) {
	user, err := users.GetUser(message.Author)
	if err != nil {
		return nil, err
	}
	if !posts.GetMembership().IsAllowed(user.GetRights()) {
		return user, errors.New("permission denied for posts")
	}
	post, err := posts.GetQueries().GetVocal(message.Post)
	if err != nil {
		return nil, err
	}
	if !post.GetMembership().IsAllowed(user.GetRights()) {
		return user, errors.New("permission denied for post")
	}
	return user, nil
}

func storeMessage(message *au10.PublisherMessageData, db postdb.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = tx.AppendMessage(uint32(message.Post), uint32(message.ID), message.Data)
	if err != nil {
		return err
	}
	return tx.Commit()
}
