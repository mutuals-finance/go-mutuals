package emails

import (
	"context"
	"testing"
	"time"

	"github.com/mikeydub/go-gallery/db/gen/coredb"
)

func TestNotificationTemplating_Success(t *testing.T) {
	a, _, pgx := setupTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	q := coredb.New(pgx)

	t.Run("creates a template for follow notifications", func(t *testing.T) {
		data, err := notifToTemplateData(ctx, q, followNotif)
		a.NoError(err)
		a.Equal(testUser2.Username.String, data.Actor)
	})

	t.Run("creates a template for view notifications", func(t *testing.T) {
		data, err := notifToTemplateData(ctx, q, viewNotif)
		a.NoError(err)
		a.Equal(testUser2.Username.String, data.Actor)
	})

}
