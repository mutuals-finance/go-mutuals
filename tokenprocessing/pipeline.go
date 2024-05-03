package tokenprocessing

import (
	"cloud.google.com/go/storage"
	"context"
	"errors"
	"fmt"
	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/media"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/service/tracing"
	"github.com/SplitFi/go-splitfi/util"
	"github.com/everFinance/goar"
	shell "github.com/ipfs/go-ipfs-api"
	"github.com/jackc/pgtype"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

type tokenProcessor struct {
	queries        *db.Queries
	httpClient     *http.Client
	metadataFinder *MetadataFinder
	ipfsClient     *shell.Shell
	arweaveClient  *goar.Client
	stg            *storage.Client
	tokenBucket    string
}

func NewTokenProcessor(queries *db.Queries, httpClient *http.Client, metadataFinder *MetadataFinder, ipfsClient *shell.Shell, arweaveClient *goar.Client, stg *storage.Client, tokenBucket string) *tokenProcessor {
	return &tokenProcessor{
		queries:        queries,
		metadataFinder: metadataFinder,
		httpClient:     httpClient,
		ipfsClient:     ipfsClient,
		arweaveClient:  arweaveClient,
		stg:            stg,
		tokenBucket:    tokenBucket,
	}
}

type tokenProcessingJob struct {
	tp               *tokenProcessor
	id               persist.DBID
	token            persist.TokenIdentifiers
	contract         persist.ContractIdentifiers
	cause            persist.ProcessingCause
	pipelineMetadata *persist.PipelineMetadata
	// profileImageKey is an optional key in the metadata that the pipeline should also process as a profile image.
	// The pipeline only looks at the root level of the metadata for the key and will also not fail if the key is missing
	// or if processing media for the key fails.
	profileImageKey string
	// refreshMetadata is an optional flag that indicates that the pipeline should check for new metadata when enabled
	refreshMetadata bool
	// startingMetadata is starting metadata to use to process media from. If empty or refreshMetadata is set, then the pipeline will try to get new metadata.
	startingMetadata persist.TokenMetadata
	// isSpamJob indicates that the job is processing a spam token. It's currently used to exclude events from Sentry.
	isSpamJob bool
	// isFxhash indicates that the job is processing a fxhash token.
	isFxhash bool
	// requireImage indicates that the pipeline should return an error if an image URL is present but an image wasn't cached.
	requireImage bool
	// requireFxHashSigned indicates that the pipeline should return an error if the token is FxHash but it isn't signed yet.
	requireFxHashSigned bool
	//fxHashIsSignedF is called to determine if a token is signed by Fxhash. It's currently used to determine if the token should be retried at a later time if it is not signed yet.
	fxHashIsSignedF func(persist.TokenMetadata) bool
	// imgKeywords are fields in the token's metadata that the pipeline should treat as images. If imgKeywords is empty, the chain's default keywords are used instead.
	imgKeywords []string
	// animKeywords are fields in the token's metadata that the pipeline should treat as animations. If animKeywords is empty, the chain's default keywords are used instead.
	animKeywords []string
	// placeHolderImageURL is an image URL that is downloaded from if processing from metadata fails
	placeHolderImageURL string
}

type PipelineOption func(*tokenProcessingJob)

type pOpts struct{}

var PipelineOpts pOpts

func (pOpts) WithProfileImageKey(key string) PipelineOption {
	return func(j *tokenProcessingJob) {
		j.profileImageKey = key
	}
}

func (pOpts) WithRefreshMetadata() PipelineOption {
	return func(j *tokenProcessingJob) {
		j.refreshMetadata = true
	}
}

func (pOpts) WithMetadata(t persist.TokenMetadata) PipelineOption {
	return func(j *tokenProcessingJob) {
		j.startingMetadata = t
	}
}

func (pOpts) WithIsSpamJob(isSpamJob bool) PipelineOption {
	return func(j *tokenProcessingJob) {
		j.isSpamJob = isSpamJob
	}
}

func (pOpts) WithRequireImage() PipelineOption {
	return func(j *tokenProcessingJob) {
		j.requireImage = true
	}
}

func (pOpts) WithRequireProhibitionimage(c db.Contract) PipelineOption {
	return func(j *tokenProcessingJob) {
		if platform.IsProhibition(c.Chain, c.Address) {
			j.requireImage = true
		}
	}
}

func (pOpts) WithIsFxhash(isFxhash bool) PipelineOption {
	return func(j *tokenProcessingJob) {
		j.isFxhash = isFxhash
	}
}

func (pOpts) WithRequireFxHashSigned(td db.TokenDefinition, c db.Contract) PipelineOption {
	return func(j *tokenProcessingJob) {
		if td.IsFxhash {
			j.isFxhash = true
			j.requireFxHashSigned = true
			j.fxHashIsSignedF = func(m persist.TokenMetadata) bool { return platform.IsFxhashSigned(td, c, m) }
		}
	}
}

func (pOpts) WithKeywords(td db.TokenDefinition) PipelineOption {
	return func(j *tokenProcessingJob) {
		j.imgKeywords, j.animKeywords = platform.KeywordsFor(td)
	}
}

func (pOpts) WithPlaceholderImageURL(u string) PipelineOption {
	return func(j *tokenProcessingJob) {
		j.placeHolderImageURL = u
	}
}

type ErrImageResultRequired struct{ Err error }

func (e ErrImageResultRequired) Unwrap() error { return e.Err }
func (e ErrImageResultRequired) Error() string {
	msg := "failed to process required image"
	if e.Err != nil {
		msg += ": " + e.Err.Error()
	}
	return msg
}

// ErrRequiredSignedToken indicates that the token isn't signed
var ErrRequiredSignedToken = errors.New("token isn't signed")

func (tp *tokenProcessor) ProcessToken(ctx context.Context, token persist.TokenIdentifiers, contract persist.ContractIdentifiers, cause persist.ProcessingCause, opts ...PipelineOption) (db.TokenMedia, error) {
	runID := persist.GenerateID()

	ctx = logger.NewContextWithFields(ctx, logrus.Fields{"runID": runID})

	job := &tokenProcessingJob{
		id:               runID,
		tp:               tp,
		token:            token,
		contract:         contract,
		cause:            cause,
		pipelineMetadata: new(persist.PipelineMetadata),
	}

	for _, opt := range opts {
		opt(job)
	}

	if len(job.imgKeywords) == 0 {
		k, _ := token.Chain.BaseKeywords()
		job.imgKeywords = k
	}

	if len(job.animKeywords) == 0 {
		_, k := token.Chain.BaseKeywords()
		job.animKeywords = k
	}

	media, err := job.Run(ctx)

	if err != nil {
		reportJobError(ctx, err, *job)
	}

	return media, err
}

// Run runs the pipeline, returning the media that was created by the run.
func (tpj *tokenProcessingJob) Run(ctx context.Context) (db.TokenMedia, error) {
	span, ctx := tracing.StartSpan(ctx, "pipeline.run", fmt.Sprintf("run %s", tpj.id))
	defer tracing.FinishSpan(span)

	logger.For(ctx).Infof("starting token processing pipeline for token %s", tpj.token.String())

	mediaCtx, cancel := context.WithTimeout(ctx, time.Minute*10)
	defer cancel()

	metadata, mediaErr := tpj.createMediaForToken(mediaCtx)

	saved, err := tpj.persistResults(ctx, metadata)
	if err != nil {
		return saved, err
	}

	return saved, mediaErr
}

func wrapWithBadTokenErr(err error) error {
	if errors.Is(err, media.ErrNoMediaURLs) || util.ErrorIs[errInvalidMedia](err) || util.ErrorIs[errNoDataFromReader](err) || errors.Is(err, ErrRequiredSignedToken) {
		err = tokenmanage.ErrBadToken{Err: err}
	}
	return err
}

func (tpj *tokenProcessingJob) createMediaForToken(ctx context.Context) (persist.TokenMetadata, error) {
	var (
		metadata persist.TokenMetadata
		err      error
	)

	func() {
		metadataCallback, ctx := persist.TrackStepStatus(ctx, &tpj.pipelineMetadata.MetadataRetrieval, "MetadataRetrieval")
		defer metadataCallback()

		metadata, err = tpj.tp.metadataFinder.GetMetadata(ctx, tpj.token)
		if err != nil {
			return
		}

		// use the starting metadata
		metadata = tpj.startingMetadata
	}()

	if err != nil {
		return metadata, wrapWithBadTokenErr(err)
	}

	return metadata, err
}

func toJSONB(v any) (pgtype.JSONB, error) {
	var j pgtype.JSONB
	err := j.Set(v)
	return j, err
}

func (tpj *tokenProcessingJob) persistResults(ctx context.Context, metadata persist.TokenMetadata) (db.TokenMedia, error) {
	newMetadata, err := toJSONB(metadata)
	if err != nil {
		return db.TokenMedia{}, err
	}

	name, description := findNameAndDescription(metadata)

	params := db.InsertTokenPipelineResultsParams{
		ProcessingJobID:  tpj.id,
		PipelineMetadata: *tpj.pipelineMetadata,
		ProcessingCause:  tpj.cause,
		ProcessorVersion: "",
		RetiringMediaID:  persist.GenerateID(),
		Chain:            tpj.token.Chain,
		ContractAddress:  tpj.contract.ContractAddress,
		TokenID:          tpj.token.TokenID,
		NewMetadata:      newMetadata,
		NewName:          util.ToNullString(name, true),
		NewDescription:   util.ToNullString(description, true),
	}

	params.TokenProperties = persist.TokenProperties{
		HasMetadata:     len(metadata) > 0,
		HasPrimaryMedia: media.MediaType.IsValid() && media.MediaURL != "",
		HasThumbnail:    media.ThumbnailURL != "",
		HasLiveRender:   media.LivePreviewURL != "",
		HasDimensions:   media.Dimensions.Valid(),
		HasName:         params.NewName.String != "",
		HasDescription:  params.NewDescription.String != "",
	}

	r, err := tpj.tp.queries.InsertTokenPipelineResults(ctx, params)
	return r.TokenMedia, err
}
