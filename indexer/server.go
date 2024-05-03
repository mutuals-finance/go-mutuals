package indexer

import (
	"context"
	"github.com/SplitFi/go-splitfi/db/gen/indexerdb"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func getStatus(i *indexer, queries *indexerdb.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c, 10*time.Second)
		defer cancel()

		mostRecent, _ := queries.MostRecentBlock(ctx)

		c.JSON(http.StatusOK, gin.H{
			"most_recent_blockchain": i.mostRecentBlock,
			"most_recent_db":         mostRecent,
			"last_synced_chunk":      i.lastSyncedChunk,
			"is_listening":           i.isListening,
		})
	}
}
