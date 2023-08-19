package media

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/SplitFi/go-splitfi/env"
	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/mediamapper"
	sentryutil "github.com/SplitFi/go-splitfi/service/sentry"
	"github.com/SplitFi/go-splitfi/service/tracing"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"

	"cloud.google.com/go/storage"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/service/rpc"
	"github.com/SplitFi/go-splitfi/util"
	"github.com/everFinance/goar"
	shell "github.com/ipfs/go-ipfs-api"
	htransport "google.golang.org/api/transport/http"
)

func init() {
	env.RegisterValidation("IPFS_URL", "required")
}

var errAlreadyHasMedia = errors.New("token already has preview and thumbnail URLs")

type Keywords interface {
	ForToken(contract persist.Address) []string
}

type DefaultKeywords []string

type errUnsupportedURL struct {
	url string
}

type errUnsupportedMediaType struct {
	mediaType persist.MediaType
}

type errNoDataFromReader struct {
	err error
	url string
}

func (e errNoDataFromReader) Error() string {
	return fmt.Sprintf("no data from reader: %s (url: %s)", e.err, e.url)
}

type mediaWithContentType struct {
	mediaType   persist.MediaType
	contentType string
}

var postfixesToMediaTypes = map[string]mediaWithContentType{
	"jpg":  {persist.MediaTypeImage, "image/jpeg"},
	"jpeg": {persist.MediaTypeImage, "image/jpeg"},
	"png":  {persist.MediaTypeImage, "image/png"},
	"webp": {persist.MediaTypeImage, "image/webp"},
	"gif":  {persist.MediaTypeGIF, "image/gif"},
	"mp4":  {persist.MediaTypeVideo, "video/mp4"},
	"webm": {persist.MediaTypeVideo, "video/webm"},
	"glb":  {persist.MediaTypeAnimation, "model/gltf-binary"},
	"gltf": {persist.MediaTypeAnimation, "model/gltf+json"},
	"svg":  {persist.MediaTypeImage, "image/svg+xml"},
	"pdf":  {persist.MediaTypePDF, "application/pdf"},
	"html": {persist.MediaTypeHTML, "text/html"},
}

func NewStorageClient(ctx context.Context) *storage.Client {
	opts := append([]option.ClientOption{}, option.WithScopes([]string{storage.ScopeFullControl}...))

	if env.GetString("ENV") == "local" {
		fi, err := util.LoadEncryptedServiceKeyOrError("./secrets/dev/service-key-dev.json")
		if err != nil {
			logger.For(ctx).WithError(err).Error("failed to find service key file (local), running without storage client")
			return nil
		}
		opts = append(opts, option.WithCredentialsJSON(fi))
	}

	transport, err := htransport.NewTransport(ctx, tracing.NewTracingTransport(http.DefaultTransport, false), opts...)
	if err != nil {
		panic(err)
	}

	client, _, err := htransport.NewClient(ctx)
	if err != nil {
		panic(err)
	}
	client.Transport = transport

	storageClient, err := storage.NewClient(ctx, option.WithHTTPClient(client))
	if err != nil {
		panic(err)
	}

	return storageClient
}

// MakePreviewsForMetadata uses a metadata map to generate media content and cache resized versions of the media content.
func MakePreviewsForMetadata(pCtx context.Context, metadata persist.TokenMetadata, contractAddress persist.Address, tokenURI string, chain persist.Chain, ipfsClient *shell.Shell, arweaveClient *goar.Client, storageClient *storage.Client, tokenBucket string, imageKeywords, animationKeywords Keywords) (persist.Media, error) {
	name := fmt.Sprintf("%s-%s", chain, contractAddress)
	imgURL, vURL := FindImageAndAnimationURLs(pCtx, contractAddress, metadata, tokenURI, animationKeywords, imageKeywords, true)
	logger.For(pCtx).Infof("got imgURL=%s;videoURL=%s", imgURL, vURL)

	var (
		imgCh, vidCh         chan cacheResult
		imgResult, vidResult cacheResult
		mediaType            persist.MediaType
		res                  persist.Media
	)

	tids := persist.NewChainAddress(contractAddress, chain)
	if vURL != "" {
		vidCh = downloadMediaFromURL(pCtx, tids, storageClient, arweaveClient, ipfsClient, "video", vURL, name, tokenBucket)
	}
	if imgURL != "" {
		imgCh = downloadMediaFromURL(pCtx, tids, storageClient, arweaveClient, ipfsClient, "image", imgURL, name, tokenBucket)
	}

	if vidCh != nil {
		vidResult = <-vidCh
	}
	if imgCh != nil {
		imgResult = <-imgCh
	}

	// Neither download worked
	if (vidResult.err != nil && vidResult.mediaType == "") && (imgResult.err != nil && imgResult.mediaType == "") {
		return persist.Media{}, vidResult.err // Just use the video error
	}

	if imgResult.mediaType != "" {
		mediaType = imgResult.mediaType
	}
	if vidResult.mediaType != "" {
		mediaType = vidResult.mediaType
	}

	if asString, ok := metadata["media_type"].(string); !mediaType.IsValid() && ok && asString != "" {
		mediaType = persist.MediaType(RawFormatToMediaType(asString))
	}

	if asString, ok := metadata["format"].(string); !mediaType.IsValid() && ok && asString != "" {
		mediaType = persist.MediaType(RawFormatToMediaType(asString))
	}

	pCtx = logger.NewContextWithFields(pCtx, logrus.Fields{"mediaType": mediaType})
	logger.For(pCtx).Infof("using '%s' as the mediaType", mediaType)

	wg := &sync.WaitGroup{}

	// if nothing was cached in the image step and the image step did process an image type, delete the now stale cached image
	if !imgResult.cached && imgResult.mediaType.IsImageLike() {
		logger.For(pCtx).Debug("imgResult not cached, deleting cached version if any")
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			deleteMedia(ctx, tokenBucket, fmt.Sprintf("image-%s", name), storageClient)
		}()
	}

	// if nothing was cached in the image step and the image step did process an image type, delete the now stale cached live render
	if !imgResult.cached && imgResult.mediaType.IsAnimationLike() {
		logger.For(pCtx).Debug("imgResult not cached, deleting cached version if any")
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			deleteMedia(ctx, tokenBucket, fmt.Sprintf("liverender-%s", name), storageClient)
		}()
	}
	// if nothing was cached in the video step and the video step did process a video type, delete the now stale cached video
	if !vidResult.cached && vidResult.mediaType.IsAnimationLike() {
		logger.For(pCtx).Debug("vidResult not cached, deleting cached version if any")
		wg.Add(2)

		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			deleteMedia(ctx, tokenBucket, fmt.Sprintf("video-%s", name), storageClient)
		}()
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			deleteMedia(ctx, tokenBucket, fmt.Sprintf("liverender-%s", name), storageClient)
		}()
	}

	// if something was cached but neither media type is animation type, we can assume that there was nothing thumbnailed therefore any thumbnail or liverender is stale
	if (imgResult.cached || vidResult.cached) && (!imgResult.mediaType.IsAnimationLike() && !vidResult.mediaType.IsAnimationLike()) {
		logger.For(pCtx).Debug("neither cached, deleting thumbnail if any")
		wg.Add(2)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			deleteMedia(ctx, tokenBucket, fmt.Sprintf("thumbnail-%s", name), storageClient)
		}()
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			deleteMedia(ctx, tokenBucket, fmt.Sprintf("liverender-%s", name), storageClient)
		}()
	}

	// imgURL does not work, but vidURL does, don't try to use imgURL
	if _, ok := imgResult.err.(errNoDataFromReader); ok && (vidResult.cached && vidResult.mediaType.IsAnimationLike()) {
		imgURL = ""
	}

	logger.For(pCtx).Debug("waiting for all necessary media to be deleted")
	wg.Wait()
	logger.For(pCtx).Debug("deleting finished")

	switch mediaType {
	case persist.MediaTypeImage:
		res = getImageMedia(pCtx, name, tokenBucket, storageClient, vURL, imgURL)
	case persist.MediaTypeVideo, persist.MediaTypeAudio, persist.MediaTypeText, persist.MediaTypePDF, persist.MediaTypeAnimation:
		res = getAuxilaryMedia(pCtx, name, tokenBucket, storageClient, vURL, imgURL, mediaType)
	case persist.MediaTypeHTML:
		res = getHTMLMedia(pCtx, name, tokenBucket, storageClient, vURL, imgURL)
	case persist.MediaTypeGIF:
		res = getGIFMedia(pCtx, name, tokenBucket, storageClient, vURL, imgURL)
	case persist.MediaTypeSVG:
		res = getSvgMedia(pCtx, name, tokenBucket, storageClient, vURL, imgURL)
	default:
		res = getRawMedia(pCtx, mediaType, name, vURL, imgURL)
	}

	logger.For(pCtx).Infof("media for %s of type %s: %+v", name, mediaType, res)
	return res, nil
}

type cacheResult struct {
	mediaType persist.MediaType
	cached    bool
	err       error
}

func downloadMediaFromURL(ctx context.Context, tids persist.ChainAddress, storageClient *storage.Client, arweaveClient *goar.Client, ipfsClient *shell.Shell, urlType, mediaURL, name, bucket string) chan cacheResult {
	resultCh := make(chan cacheResult)
	ctx = logger.NewContextWithFields(ctx, logrus.Fields{
		"tokenURIType": persist.URITypeIPFS,
		"urlType":      urlType,
	})

	go func() {
		mediaType, cached, err := downloadAndCache(ctx, mediaURL, name, urlType, ipfsClient, arweaveClient, storageClient, bucket, false)
		if err == nil {
			resultCh <- cacheResult{mediaType, cached, err}
			return
		}

		switch caught := err.(type) {
		case rpc.ErrHTTP:
			if err.(rpc.ErrHTTP).Status == http.StatusNotFound {
				resultCh <- cacheResult{persist.MediaTypeInvalid, cached, err}
			} else {
				resultCh <- cacheResult{mediaType, cached, err}
			}
		case *net.DNSError:
			resultCh <- cacheResult{persist.MediaTypeInvalid, cached, err}
		case *googleapi.Error:
			panic(fmt.Errorf("googleAPI error %s: %s", caught, err))
		default:
			logger.For(ctx).Error(err)
			sentryutil.ReportError(ctx, err)
			resultCh <- cacheResult{mediaType, cached, err}
		}
	}()

	return resultCh
}

func getAuxilaryMedia(pCtx context.Context, name, tokenBucket string, storageClient *storage.Client, vURL string, imgURL string, mediaType persist.MediaType) persist.Media {
	res := persist.Media{
		MediaType: mediaType,
	}
	videoURL, err := getMediaServingURL(pCtx, tokenBucket, fmt.Sprintf("video-%s", name), storageClient)
	if err == nil {
		vURL = videoURL
	}
	imageURL := getThumbnailURL(pCtx, tokenBucket, name, imgURL, storageClient)
	if vURL != "" {
		logger.For(pCtx).Infof("using vURL %s: %s", name, vURL)
		res.MediaURL = persist.NullString(vURL)
		res.ThumbnailURL = persist.NullString(imageURL)
	} else if imageURL != "" {
		logger.For(pCtx).Infof("using imageURL for %s: %s", name, imageURL)
		res.MediaURL = persist.NullString(imageURL)
	}

	res = remapMedia(res)

	res.Dimensions, err = getMediaDimensions(pCtx, res.MediaURL.String())
	if err != nil {
		logger.For(pCtx).Errorf("failed to get dimensions for %s: %v", name, err)
	}

	if mediaType == persist.MediaTypeVideo {
		liveRenderURL, err := getMediaServingURL(pCtx, tokenBucket, fmt.Sprintf("liverender-%s", name), storageClient)
		if err != nil {
			logger.For(pCtx).Errorf("failed to get live render URL for %s: %v", name, err)
		} else {
			res.LivePreviewURL = persist.NullString(liveRenderURL)
		}
	}

	return res
}

func getGIFMedia(pCtx context.Context, name, tokenBucket string, storageClient *storage.Client, vURL string, imgURL string) persist.Media {
	res := persist.Media{
		MediaType: persist.MediaTypeGIF,
	}
	videoURL, err := getMediaServingURL(pCtx, tokenBucket, fmt.Sprintf("video-%s", name), storageClient)
	if err == nil {
		vURL = videoURL
	}
	imageURL, err := getMediaServingURL(pCtx, tokenBucket, fmt.Sprintf("image-%s", name), storageClient)
	if err == nil {
		logger.For(pCtx).Infof("found imageURL for %s: %s", name, imageURL)
		imgURL = imageURL
	}
	res.ThumbnailURL = persist.NullString(getThumbnailURL(pCtx, tokenBucket, name, imgURL, storageClient))
	if vURL != "" {
		logger.For(pCtx).Infof("using vURL %s: %s", name, vURL)
		res.MediaURL = persist.NullString(vURL)
		if imgURL != "" && res.ThumbnailURL.String() == "" {
			res.ThumbnailURL = persist.NullString(imgURL)
		}
	} else if imgURL != "" {
		logger.For(pCtx).Infof("using imgURL for %s: %s", name, imgURL)
		res.MediaURL = persist.NullString(imgURL)
	}

	res = remapMedia(res)

	res.Dimensions, err = getMediaDimensions(pCtx, res.MediaURL.String())
	if err != nil {
		logger.For(pCtx).Errorf("failed to get dimensions for %s: %v", name, err)
	}

	return res
}

func getSvgMedia(pCtx context.Context, name, tokenBucket string, storageClient *storage.Client, vURL, imgURL string) persist.Media {
	res := persist.Media{
		MediaType: persist.MediaTypeSVG,
	}
	imageURL, err := getMediaServingURL(pCtx, tokenBucket, fmt.Sprintf("svg-%s", name), storageClient)
	if err == nil {
		logger.For(pCtx).Infof("found svgURL for svg %s: %s", name, imageURL)
		res.MediaURL = persist.NullString(imageURL)
	} else {
		if vURL != "" {
			logger.For(pCtx).Infof("using vURL for svg %s: %s", name, vURL)
			res.MediaURL = persist.NullString(vURL)
			if imgURL != "" {
				res.ThumbnailURL = persist.NullString(imgURL)
			}
		} else if imgURL != "" {
			logger.For(pCtx).Infof("using imgURL for svg %s: %s", name, imgURL)
			res.MediaURL = persist.NullString(imgURL)
		}
	}

	res = remapMedia(res)

	res.Dimensions, err = getSvgDimensions(pCtx, res.MediaURL.String())
	if err != nil {
		logger.For(pCtx).Errorf("failed to get dimensions for svg %s: %v", name, err)
	}

	return res
}

type svgDimensions struct {
	XMLName xml.Name `xml:"svg"`
	Width   string   `xml:"width,attr"`
	Height  string   `xml:"height,attr"`
	Viewbox string   `xml:"viewBox,attr"`
}

func getSvgDimensions(ctx context.Context, url string) (persist.Dimensions, error) {
	buf := &bytes.Buffer{}
	if strings.HasPrefix(url, "http") {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return persist.Dimensions{}, err
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return persist.Dimensions{}, err
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return persist.Dimensions{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		_, err = io.Copy(buf, resp.Body)
		if err != nil {
			return persist.Dimensions{}, err
		}
	} else {
		buf = bytes.NewBufferString(url)
	}

	if bytes.HasSuffix(buf.Bytes(), []byte(`<!-- Generated by SVGo -->`)) {
		buf = bytes.NewBuffer(bytes.TrimSuffix(buf.Bytes(), []byte(`<!-- Generated by SVGo -->`)))
	}

	var s svgDimensions
	if err := xml.NewDecoder(buf).Decode(&s); err != nil {
		return persist.Dimensions{}, err
	}

	if (s.Width == "" || s.Height == "") && s.Viewbox == "" {
		return persist.Dimensions{}, fmt.Errorf("no dimensions found for %s", url)
	}

	if s.Viewbox != "" {
		parts := strings.Split(s.Viewbox, " ")
		if len(parts) != 4 {
			return persist.Dimensions{}, fmt.Errorf("invalid viewbox for %s", url)
		}
		s.Width = parts[2]
		s.Height = parts[3]

	}

	width, err := strconv.Atoi(s.Width)
	if err != nil {
		return persist.Dimensions{}, err
	}

	height, err := strconv.Atoi(s.Height)
	if err != nil {
		return persist.Dimensions{}, err
	}

	return persist.Dimensions{
		Width:  width,
		Height: height,
	}, nil
}

func getImageMedia(pCtx context.Context, name, tokenBucket string, storageClient *storage.Client, vURL, imgURL string) persist.Media {
	res := persist.Media{
		MediaType: persist.MediaTypeImage,
	}
	imageURL, err := getMediaServingURL(pCtx, tokenBucket, fmt.Sprintf("image-%s", name), storageClient)
	if err == nil {
		logger.For(pCtx).Infof("found imageURL for %s: %s", name, imageURL)
		res.MediaURL = persist.NullString(imageURL)
	} else {
		if vURL != "" {
			logger.For(pCtx).Infof("using vURL for %s: %s", name, vURL)
			res.MediaURL = persist.NullString(vURL)
			if imgURL != "" {
				res.ThumbnailURL = persist.NullString(imgURL)
			}
		} else if imgURL != "" {
			logger.For(pCtx).Infof("using imgURL for %s: %s", name, imgURL)
			res.MediaURL = persist.NullString(imgURL)
		}
	}

	res = remapMedia(res)

	res.Dimensions, err = getMediaDimensions(pCtx, res.MediaURL.String())
	if err != nil {
		logger.For(pCtx).Errorf("failed to get dimensions for %s: %v", name, err)
	}

	return res
}

func getHTMLMedia(pCtx context.Context, name, tokenBucket string, storageClient *storage.Client, vURL, imgURL string) persist.Media {
	res := persist.Media{
		MediaType: persist.MediaTypeHTML,
	}

	videoURL, err := getMediaServingURL(pCtx, tokenBucket, fmt.Sprintf("video-%s", name), storageClient)
	if err == nil {
		vURL = videoURL
	}
	if vURL != "" {
		logger.For(pCtx).Infof("using vURL for %s: %s", name, vURL)
		res.MediaURL = persist.NullString(vURL)
	} else if imgURL != "" {
		logger.For(pCtx).Infof("using imgURL for %s: %s", name, imgURL)
		res.MediaURL = persist.NullString(imgURL)
	}
	res.ThumbnailURL = persist.NullString(getThumbnailURL(pCtx, tokenBucket, name, imgURL, storageClient))

	res = remapMedia(res)

	dimensions, err := getHTMLDimensions(pCtx, res.MediaURL.String())
	if err != nil {
		logger.For(pCtx).Errorf("failed to get dimensions for %s: %v", name, err)
	}

	res.Dimensions = dimensions

	return res
}

type iframeDimensions struct {
	XMLName xml.Name `xml:"iframe"`
	Width   string   `xml:"width,attr"`
	Height  string   `xml:"height,attr"`
}

func getHTMLDimensions(ctx context.Context, url string) (persist.Dimensions, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return persist.Dimensions{}, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return persist.Dimensions{}, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return persist.Dimensions{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var s iframeDimensions
	if err := xml.NewDecoder(resp.Body).Decode(&s); err != nil {
		return persist.Dimensions{}, err
	}

	if s.Width == "" || s.Height == "" {
		return persist.Dimensions{}, fmt.Errorf("no dimensions found for %s", url)
	}

	width, err := strconv.Atoi(s.Width)
	if err != nil {
		return persist.Dimensions{}, err
	}

	height, err := strconv.Atoi(s.Height)
	if err != nil {
		return persist.Dimensions{}, err
	}

	return persist.Dimensions{
		Width:  width,
		Height: height,
	}, nil

}

func getRawMedia(pCtx context.Context, mediaType persist.MediaType, name, vURL, imgURL string) persist.Media {
	var res persist.Media
	res.MediaType = mediaType
	if vURL != "" {
		logger.For(pCtx).Infof("using vURL for %s: %s", name, vURL)
		res.MediaURL = persist.NullString(vURL)
		if imgURL != "" {
			res.ThumbnailURL = persist.NullString(imgURL)
		}
	} else if imgURL != "" {
		logger.For(pCtx).Infof("using imgURL for %s: %s", name, imgURL)
		res.MediaURL = persist.NullString(imgURL)
	}

	res = remapMedia(res)

	dimensions, err := getMediaDimensions(pCtx, res.MediaURL.String())
	if err != nil {
		logger.For(pCtx).Errorf("failed to get dimensions for %s: %v", name, err)
	}
	res.Dimensions = dimensions
	return res
}

func remapPaths(mediaURL string) string {
	switch persist.URITypeIPFS {
	case persist.URITypeIPFS, persist.URITypeIPFSAPI:
		path := util.GetURIPath(mediaURL, false)
		return fmt.Sprintf("%s/ipfs/%s", env.GetString("IPFS_URL"), path)
	case persist.URITypeArweave:
		path := util.GetURIPath(mediaURL, false)
		return fmt.Sprintf("https://arweave.net/%s", path)
	default:
		return mediaURL
	}

}

func remapMedia(media persist.Media) persist.Media {
	media.MediaURL = persist.NullString(remapPaths(strings.TrimSpace(media.MediaURL.String())))
	media.ThumbnailURL = persist.NullString(remapPaths(strings.TrimSpace(media.ThumbnailURL.String())))
	return media
}

func FindImageAndAnimationURLs(ctx context.Context, contractAddress persist.Address, metadata persist.TokenMetadata, tokenURI string, animationKeywords, imageKeywords Keywords, predict bool) (imgURL string, vURL string) {
	ctx = logger.NewContextWithFields(ctx, logrus.Fields{"contractAddress": contractAddress})
	if metaMedia, ok := metadata["media"].(map[string]any); ok {
		logger.For(ctx).Debugf("found media metadata: %s", metaMedia)
		var mediaType persist.MediaType

		if mime, ok := metaMedia["mimeType"].(string); ok {
			mediaType = persist.MediaFromContentType(mime)
		}
		if uri, ok := metaMedia["uri"].(string); ok {
			switch mediaType {
			case persist.MediaTypeImage, persist.MediaTypeSVG, persist.MediaTypeGIF:
				imgURL = uri
			default:
				vURL = uri
			}
		}
	}

	if imgURL == "" && vURL == "" {
		logger.For(ctx).Debugf("no image url found, using token URI: %s", tokenURI)
	}

	if predict {
		return predictTrueURLs(ctx, imgURL, vURL)
	}
	return imgURL, vURL

}

func FindNameAndDescription(ctx context.Context, metadata persist.TokenMetadata) (string, string) {
	name, ok := util.GetValueFromMapUnsafe(metadata, "name", util.DefaultSearchDepth).(string)
	if !ok {
		name = ""
	}

	description, ok := util.GetValueFromMapUnsafe(metadata, "description", util.DefaultSearchDepth).(string)
	if !ok {
		description = ""
	}

	return name, description
}

func predictTrueURLs(ctx context.Context, curImg, curV string) (string, string) {
	imgMediaType, _, _, err := PredictMediaType(ctx, curImg)
	if err != nil {
		return curImg, curV
	}
	vMediaType, _, _, err := PredictMediaType(ctx, curV)
	if err != nil {
		return curImg, curV
	}

	if imgMediaType.IsAnimationLike() && !vMediaType.IsAnimationLike() {
		return curV, curImg
	}

	if !imgMediaType.IsValid() || !vMediaType.IsValid() {
		return curImg, curV
	}

	if imgMediaType.IsMorePriorityThan(vMediaType) {
		return curV, curImg
	}

	return curImg, curV
}

func getThumbnailURL(pCtx context.Context, tokenBucket string, name string, imgURL string, storageClient *storage.Client) string {
	if storageImageURL, err := getMediaServingURL(pCtx, tokenBucket, fmt.Sprintf("image-%s", name), storageClient); err == nil {
		logger.For(pCtx).Infof("found imageURL for thumbnail %s: %s", name, storageImageURL)
		return storageImageURL
	} else if storageImageURL, err = getMediaServingURL(pCtx, tokenBucket, fmt.Sprintf("svg-%s", name), storageClient); err == nil {
		logger.For(pCtx).Infof("found svg for thumbnail %s: %s", name, storageImageURL)
		return storageImageURL
	} else if imgURL != "" {
		logger.For(pCtx).Infof("using imgURL for thumbnail %s: %s", name, imgURL)
		return imgURL
	} else if storageImageURL, err := getMediaServingURL(pCtx, tokenBucket, fmt.Sprintf("thumbnail-%s", name), storageClient); err == nil {
		logger.For(pCtx).Infof("found thumbnailURL for %s: %s", name, storageImageURL)
		return storageImageURL
	}
	return ""
}

func objectExists(ctx context.Context, client *storage.Client, bucket, fileName string) (bool, error) {
	objHandle := client.Bucket(bucket).Object(fileName)
	_, err := objHandle.Attrs(ctx)
	if err != nil && err != storage.ErrObjectNotExist {
		return false, fmt.Errorf("could not get object attrs for %s: %s", objHandle.ObjectName(), err)
	}
	return err != storage.ErrObjectNotExist, nil
}

func purgeIfExists(ctx context.Context, bucket string, fileName string, client *storage.Client) error {
	exists, err := objectExists(ctx, client, bucket, fileName)
	if err != nil {
		return err
	}
	if exists {
		if err := mediamapper.PurgeImage(ctx, fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucket, fileName)); err != nil {
			logger.For(ctx).WithError(err).Errorf("could not purge file %s", fileName)
		}
	}

	return nil
}

func persistToStorage(ctx context.Context, client *storage.Client, reader io.Reader, bucket, fileName, contentType string) error {
	writer := newObjectWriter(ctx, client, bucket, fileName, contentType)
	if _, err := io.Copy(writer, reader); err != nil {
		return fmt.Errorf("could not write to bucket %s for %s: %s", bucket, fileName, err)
	}
	return writer.Close()
}

func cacheRawMedia(ctx context.Context, reader io.Reader, bucket, fileName string, contentType string, client *storage.Client) error {
	err := persistToStorage(ctx, client, reader, bucket, fileName, contentType)
	go purgeIfExists(context.Background(), bucket, fileName, client)
	return err
}

func cacheRawSvgMedia(ctx context.Context, reader io.Reader, bucket, name string, client *storage.Client) error {
	return cacheRawMedia(ctx, reader, bucket, fmt.Sprintf("svg-%s", name), "image/svg+xml", client)
}

func cacheRawVideoMedia(ctx context.Context, reader io.Reader, bucket, name, contentType string, client *storage.Client) error {
	return cacheRawMedia(ctx, reader, bucket, fmt.Sprintf("video-%s", name), contentType, client)
}

func cacheRawImageMedia(ctx context.Context, reader io.Reader, bucket, name, contentType string, client *storage.Client) error {
	return cacheRawMedia(ctx, reader, bucket, fmt.Sprintf("image-%s", name), contentType, client)
}

func cacheRawAnimationMedia(ctx context.Context, reader io.Reader, bucket, fileName string, client *storage.Client) error {
	sw := newObjectWriter(ctx, client, bucket, fileName, "")
	writer := gzip.NewWriter(sw)

	_, err := io.Copy(writer, reader)
	if err != nil {
		return fmt.Errorf("could not write to bucket %s for %s: %s", bucket, fileName, err)
	}

	if err := writer.Close(); err != nil {
		return err
	}

	if err := sw.Close(); err != nil {
		return err
	}

	go purgeIfExists(context.Background(), bucket, fileName, client)
	return nil
}

func thumbnailAndCache(ctx context.Context, videoURL, bucket, name string, client *storage.Client) error {

	fileName := fmt.Sprintf("thumbnail-%s", name)
	logger.For(ctx).Infof("caching thumbnail for '%s'", fileName)

	timeBeforeCopy := time.Now()

	sw := newObjectWriter(ctx, client, bucket, fileName, "image/jpeg")

	logger.For(ctx).Infof("thumbnailing %s", videoURL)
	if err := thumbnailVideoToWriter(ctx, videoURL, sw); err != nil {
		return fmt.Errorf("could not thumbnail to bucket %s for '%s': %s", bucket, fileName, err)
	}

	if err := sw.Close(); err != nil {
		return err
	}

	logger.For(ctx).Infof("storage copy took %s", time.Since(timeBeforeCopy))

	go purgeIfExists(context.Background(), bucket, fileName, client)

	return nil
}

func createLiveRenderAndCache(ctx context.Context, videoURL, bucket, name string, client *storage.Client) error {

	fileName := fmt.Sprintf("liverender-%s", name)
	logger.For(ctx).Infof("caching live render media for '%s'", fileName)

	timeBeforeCopy := time.Now()

	sw := newObjectWriter(ctx, client, bucket, fileName, "video/mp4")

	logger.For(ctx).Infof("creating live render for %s", videoURL)
	if err := createLiveRenderPreviewVideo(ctx, videoURL, sw); err != nil {
		return fmt.Errorf("could not live render to bucket %s for '%s': %s", bucket, fileName, err)
	}

	if err := sw.Close(); err != nil {
		return err
	}

	logger.For(ctx).Infof("storage copy took %s", time.Since(timeBeforeCopy))

	go purgeIfExists(context.Background(), bucket, fileName, client)

	return nil
}

func deleteMedia(ctx context.Context, bucket, fileName string, client *storage.Client) error {
	return client.Bucket(bucket).Object(fileName).Delete(ctx)
}

func getMediaServingURL(pCtx context.Context, bucketID, objectID string, client *storage.Client) (string, error) {
	if exists, err := objectExists(pCtx, client, bucketID, objectID); err != nil || !exists {
		objectName := fmt.Sprintf("/gs/%s/%s", bucketID, objectID)
		return "", fmt.Errorf("failed to check if object %s exists: %s", objectName, err)
	}
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucketID, objectID), nil
}

func downloadAndCache(pCtx context.Context, mediaURL, name, ipfsPrefix string, ipfsClient *shell.Shell, arweaveClient *goar.Client, storageClient *storage.Client, bucket string, isRecursive bool) (persist.MediaType, bool, error) {
	asURI := mediaURL
	timeBeforePredict := time.Now()
	mediaType, contentType, contentLength, _ := PredictMediaType(pCtx, asURI)
	pCtx = logger.NewContextWithFields(pCtx, logrus.Fields{
		"mediaType":   mediaType,
		"contentType": contentType,
	})
	logger.For(pCtx).Infof("predicted media type from '%s' as '%s' with length %s in %s", mediaURL, mediaType, util.InByteSizeFormat(uint64(contentLength)), time.Since(timeBeforePredict))

outer:
	switch mediaType {
	case persist.MediaTypeVideo, persist.MediaTypeUnknown, persist.MediaTypeSVG, persist.MediaTypeBase64BMP:
		break outer
	default:
		switch persist.URITypeIPFS {
		case persist.URITypeIPFS, persist.URITypeArweave:
			logger.For(pCtx).Infof("uri for '%s' is of type '%s', trying to cache", name, persist.URITypeIPFS)
			break outer
		default:
			logger.For(pCtx).Infof("skipping caching of mediaType '%s' and uriType '%s'", mediaType, persist.URITypeIPFS)
			return mediaType, false, nil
		}
	}

	timeBeforeDataReader := time.Now()
	reader, err := rpc.GetDataFromURIAsReader(pCtx, asURI, ipfsClient, arweaveClient)
	if err != nil {
		return mediaType, false, errNoDataFromReader{err: err, url: mediaURL}
	}
	logger.For(pCtx).Infof("got reader for %s in %s", name, time.Since(timeBeforeDataReader))
	defer reader.Close()

	if !mediaType.IsValid() {
		timeBeforeSniff := time.Now()
		bytesToSniff, _ := reader.Headers()
		mediaType, contentType = persist.SniffMediaType(bytesToSniff)
		logger.For(pCtx).Infof("sniffed media type for %s: %s in %s", truncateString(mediaURL, 50), mediaType, time.Since(timeBeforeSniff))
	}

	switch mediaType {
	case persist.MediaTypeSVG:
		timeBeforeCache := time.Now()
		err = cacheRawSvgMedia(pCtx, reader, bucket, name, storageClient)
		if err != nil {
			return mediaType, false, err
		}
		logger.For(pCtx).Infof("cached svg for %s in %s", name, time.Since(timeBeforeCache))
		return persist.MediaTypeSVG, true, nil
	case persist.MediaTypeBase64BMP:
		timeBeforeCache := time.Now()
		err = cacheRawImageMedia(pCtx, reader, bucket, name, contentType, storageClient)
		if err != nil {
			return mediaType, false, err
		}
		logger.For(pCtx).Infof("cached image for %s in %s", name, time.Since(timeBeforeCache))
		return persist.MediaTypeImage, true, nil
	}

	switch persist.URITypeIPFS {
	case persist.URITypeIPFS, persist.URITypeArweave:
		logger.For(pCtx).Infof("caching %.2f mb of raw media with type '%s' for '%s' at '%s-%s'", float64(contentLength)/1024/1024, mediaType, mediaURL, ipfsPrefix, name)

		if mediaType == persist.MediaTypeAnimation {
			timeBeforeCache := time.Now()
			err = cacheRawAnimationMedia(pCtx, reader, bucket, fmt.Sprintf("%s-%s", ipfsPrefix, name), storageClient)
			if err != nil {
				return mediaType, false, err
			}
			logger.For(pCtx).Infof("cached animation for %s in %s", name, time.Since(timeBeforeCache))
			return mediaType, true, nil
		}
		timeBeforeCache := time.Now()
		err = cacheRawMedia(pCtx, reader, bucket, fmt.Sprintf("%s-%s", ipfsPrefix, name), contentType, storageClient)
		if err != nil {
			return mediaType, false, err
		}
		logger.For(pCtx).Infof("cached raw media for %s in %s", name, time.Since(timeBeforeCache))
		return mediaType, true, nil
	}

	return mediaType, false, nil
}

// PredictMediaType guesses the media type of the given URL.
func PredictMediaType(pCtx context.Context, url string) (persist.MediaType, string, int64, error) {

	spl := strings.Split(url, ".")
	if len(spl) > 1 {
		ext := spl[len(spl)-1]
		ext = strings.Split(ext, "?")[0]
		if t, ok := postfixesToMediaTypes[ext]; ok {
			return t.mediaType, t.contentType, 0, nil
		}
	}
	asURI := url
	uriType := persist.URITypeBase64SVG
	logger.For(pCtx).Debugf("predicting media type for %s with URI type %s", url, uriType)
	switch uriType {
	case persist.URITypeBase64JSON, persist.URITypeJSON:
		return persist.MediaTypeJSON, "application/json", int64(len(asURI)), nil
	case persist.URITypeBase64SVG, persist.URITypeSVG:
		return persist.MediaTypeSVG, "image/svg", int64(len(asURI)), nil
	case persist.URITypeBase64BMP:
		return persist.MediaTypeBase64BMP, "image/bmp", int64(len(asURI)), nil
	case persist.URITypeIPFS:
		contentType, contentLength, err := rpc.GetIPFSHeaders(pCtx, strings.TrimPrefix(asURI, "ipfs://"))
		if err != nil {
			return persist.MediaTypeUnknown, "", 0, err
		}
		return persist.MediaFromContentType(contentType), contentType, contentLength, nil
	case persist.URITypeIPFSGateway:
		contentType, contentLength, err := rpc.GetIPFSHeaders(pCtx, util.GetURIPath(asURI, false))
		if err == nil {
			return persist.MediaFromContentType(contentType), contentType, contentLength, nil
		} else if err != nil {
			logger.For(pCtx).Errorf("could not get IPFS headers for %s: %s", url, err)
		}
		fallthrough
	case persist.URITypeHTTP, persist.URITypeIPFSAPI:
		contentType, contentLength, err := rpc.GetHTTPHeaders(pCtx, url)
		if err != nil {
			return persist.MediaTypeUnknown, "", 0, err
		}
		return persist.MediaFromContentType(contentType), contentType, contentLength, nil
	}
	return persist.MediaTypeUnknown, "", 0, nil
}

func thumbnailVideoToWriter(ctx context.Context, url string, writer io.Writer) error {
	c := exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-loglevel", "error", "-i", url, "-ss", "00:00:00.000", "-vframes", "1", "-f", "mjpeg", "pipe:1")
	c.Stderr = os.Stderr
	c.Stdout = writer
	return c.Run()
}

func createLiveRenderPreviewVideo(ctx context.Context, videoURL string, writer io.Writer) error {
	c := exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-loglevel", "error", "-i", videoURL, "-ss", "00:00:00.000", "-t", "00:00:05.000", "-filter:v", "scale=720:-1", "-movflags", "frag_keyframe+empty_moov", "-c:a", "copy", "-f", "mp4", "pipe:1")
	c.Stderr = os.Stderr
	c.Stdout = writer
	return c.Run()
}

type dimensions struct {
	Streams []struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"streams"`
}

type errNoStreams struct {
	url string
	err error
}

func (e errNoStreams) Error() string {
	return fmt.Sprintf("no streams in %s: %s", e.url, e.err)
}

func getMediaDimensions(ctx context.Context, url string) (persist.Dimensions, error) {
	outBuf := &bytes.Buffer{}
	c := exec.CommandContext(ctx, "ffprobe", "-hide_banner", "-loglevel", "error", "-show_streams", url, "-print_format", "json")
	c.Stderr = os.Stderr
	c.Stdout = outBuf
	err := c.Run()
	if err != nil {
		return persist.Dimensions{}, err
	}

	var d dimensions
	err = json.Unmarshal(outBuf.Bytes(), &d)
	if err != nil {
		return persist.Dimensions{}, fmt.Errorf("failed to unmarshal ffprobe output: %w", err)
	}

	if len(d.Streams) == 0 {
		return persist.Dimensions{}, fmt.Errorf("no streams found in ffprobe output: %w", err)
	}

	dims := persist.Dimensions{}

	for _, s := range d.Streams {
		if s.Height == 0 || s.Width == 0 {
			continue
		}
		dims = persist.Dimensions{
			Width:  s.Width,
			Height: s.Height,
		}
		break
	}

	logger.For(ctx).Debugf("got dimensions %+v for %s", dims, url)
	return dims, nil
}

func truncateString(s string, i int) string {
	asRunes := []rune(s)
	if len(asRunes) > i {
		return string(asRunes[:i])
	}
	return s
}

func (d DefaultKeywords) ForToken(contract persist.Address) []string {
	return d
}

func KeywordsForChain(chain persist.Chain, imageKeywords []string, animationKeywords []string) (Keywords, Keywords) {
	return DefaultKeywords(imageKeywords), DefaultKeywords(animationKeywords)
}

func (e errUnsupportedURL) Error() string {
	return fmt.Sprintf("unsupported url %s", e.url)
}

func (e errUnsupportedMediaType) Error() string {
	return fmt.Sprintf("unsupported media type %s", e.mediaType)
}

func newObjectWriter(ctx context.Context, client *storage.Client, bucket, fileName, contentType string) *storage.Writer {
	writer := client.Bucket(bucket).Object(fileName).NewWriter(ctx)
	writer.ObjectAttrs.ContentType = contentType
	writer.CacheControl = "no-cache, no-store"
	return writer
}

func RawFormatToMediaType(format string) persist.MediaType {
	switch format {
	case "jpeg", "png", "image", "jpg", "webp":
		return persist.MediaTypeImage
	case "gif":
		return persist.MediaTypeGIF
	case "video", "mp4", "quicktime":
		return persist.MediaTypeVideo
	case "audio", "mp3", "wav":
		return persist.MediaTypeAudio
	case "pdf":
		return persist.MediaTypePDF
	case "html", "iframe":
		return persist.MediaTypeHTML
	case "svg", "svg+xml":
		return persist.MediaTypeSVG
	default:
		return persist.MediaTypeUnknown
	}
}
