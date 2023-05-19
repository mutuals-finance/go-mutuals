package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/mikeydub/go-gallery/db/gen/coredb"
	"github.com/mikeydub/go-gallery/service/persist"
	"github.com/mikeydub/go-gallery/service/persist/postgres"
	"github.com/spf13/viper"
)

// run with `go run cmd/notification_prepop/main.go ${some user ID to use as the viewer}`

func main() {

	setDefaults()

	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		fmt.Printf("Took %s", elapsed)
	}()

	ownerID := persist.DBID(os.Args[1])

	pg := postgres.NewPgxClient()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var ownerSplitID persist.DBID
	err := pg.QueryRow(ctx, "SELECT id FROM splits WHERE owner_user_id = $1", ownerID).Scan(&ownerSplitID)
	if err != nil {
		panic(err)
	}

	aFewUsers, err := pg.Query(ctx, "SELECT ID FROM USERS LIMIT 20")
	if err != nil {
		panic(err)
	}

	userIDs := make([]persist.DBID, 0)
	for aFewUsers.Next() {
		var id persist.DBID
		err := aFewUsers.Scan(&id)
		if err != nil {
			panic(err)
		}
		userIDs = append(userIDs, id)
	}

	if err != nil {
		panic(err)
	}

	notifs := make([]coredb.Notification, 0, len(userIDs))
	events := make([]coredb.Event, 0, len(userIDs))
	for _, id := range userIDs {
		action := actionForNum(rand.Intn(5))

		var resource persist.ResourceType
		var subject persist.DBID
		switch action {
		case persist.ActionViewedSplit:
			resource = persist.ResourceTypeSplit
			subject = ownerSplitID
		case persist.ActionUserFollowedUsers:
			resource = persist.ResourceTypeUser
			subject = ownerID
		}
		event := coredb.Event{
			ID:             persist.GenerateID(),
			ActorID:        persist.DBIDToNullStr(id),
			ResourceTypeID: resource,
			SubjectID:      subject,
			Action:         action,
		}

		if action == persist.ActionViewedSplit {
			event.SplitID = subject
		} else if action == persist.ActionUserFollowedUsers {
			event.UserID = subject
		}

		events = append(events, event)

		notif := coredb.Notification{
			ID:       persist.GenerateID(),
			OwnerID:  ownerID,
			Action:   action,
			EventIds: []persist.DBID{event.ID},
		}
		if action == persist.ActionViewedSplit {
			notif.SplitID = ownerSplitID
			notif.Data.AuthedViewerIDs = []persist.DBID{id}
		} else if action == persist.ActionUserFollowedUsers {
			notif.Data.FollowerIDs = []persist.DBID{id}
			randBool := rand.Intn(2) == 1
			notif.Data.FollowedBack = persist.NullBool(randBool)
		}
		notifs = append(notifs, notif)
	}

	for _, event := range events {
		if event.Action == persist.ActionViewedSplit {
			fmt.Printf("SplitID %s\n", event.SplitID)
			_, err := pg.Exec(ctx, "INSERT INTO EVENTS (ID, ACTOR_ID, RESOURCE_TYPE_ID, SUBJECT_ID, GALLERY_ID, ACTION) VALUES ($1, $2, $3, $4, $5, $6)", event.ID, event.ActorID, event.ResourceTypeID, event.SubjectID, event.SplitID, event.Action)
			if err != nil {
				panic(err)
			}
		} else if event.Action == persist.ActionUserFollowedUsers {
			fmt.Printf("UserID %s\n", event.UserID)
			_, err := pg.Exec(ctx, "INSERT INTO EVENTS (ID, ACTOR_ID, RESOURCE_TYPE_ID, SUBJECT_ID, USER_ID, ACTION) VALUES ($1, $2, $3, $4, $5, $6)", event.ID, event.ActorID, event.ResourceTypeID, event.SubjectID, event.UserID, event.Action)
			if err != nil {
				panic(err)
			}
		} else {
			_, err := pg.Exec(ctx, "INSERT INTO EVENTS (ID, ACTOR_ID, RESOURCE_TYPE_ID, SUBJECT_ID, ACTION) VALUES ($1, $2, $3, $4, $5)", event.ID, event.ActorID, event.ResourceTypeID, event.SubjectID, event.Action)
			if err != nil {
				panic(err)
			}
		}
	}

	for _, notif := range notifs {
		if notif.Action == persist.ActionViewedSplit {
			fmt.Printf("SplitID %s\n", notif.SplitID)
			_, err := pg.Exec(ctx, "INSERT INTO NOTIFICATIONS (ID, OWNER_ID, ACTION, GALLERY_ID, DATA, EVENT_IDS) VALUES ($1, $2, $3, $4, $5, $6)", notif.ID, notif.OwnerID, notif.Action, notif.SplitID, notif.Data, notif.EventIds)
			if err != nil {
				panic(err)
			}
		} else if notif.Action == persist.ActionUserFollowedUsers {
			fmt.Printf("UserID %s\n", notif.Data.FollowerIDs)
			_, err := pg.Exec(ctx, "INSERT INTO NOTIFICATIONS (ID, OWNER_ID, ACTION, DATA, EVENT_IDS) VALUES ($1, $2, $3, $4, $5)", notif.ID, notif.OwnerID, notif.Action, notif.Data, notif.EventIds)
			if err != nil {
				panic(err)
			}
		} else {
			_, err := pg.Exec(ctx, "INSERT INTO NOTIFICATIONS (ID, OWNER_ID, ACTION, DATA, EVENT_IDS) VALUES ($1, $2, $3, $4, $5)", notif.ID, notif.OwnerID, notif.Action, notif.Data, notif.EventIds)
			if err != nil {
				panic(err)
			}
		}
	}

}

func actionForNum(num int) persist.Action {
	switch num {
	case 0:
		return persist.ActionViewedSplit
	case 1:
		return persist.ActionUserFollowedUsers
	default:
		return persist.ActionViewedSplit
	}
}

func setDefaults() {
	viper.SetDefault("POSTGRES_HOST", "0.0.0.0")
	viper.SetDefault("POSTGRES_PORT", 5432)
	viper.SetDefault("POSTGRES_USER", "gallery_backend")
	viper.SetDefault("POSTGRES_PASSWORD", "")
	viper.SetDefault("POSTGRES_DB", "postgres")

	viper.AutomaticEnv()
}
