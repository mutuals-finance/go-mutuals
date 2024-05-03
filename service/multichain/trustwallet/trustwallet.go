package trustwallet

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/SplitFi/go-splitfi/env"
	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/multichain"
	"github.com/SplitFi/go-splitfi/service/persist"
	sentryutil "github.com/SplitFi/go-splitfi/service/sentry"
	"github.com/SplitFi/go-splitfi/util"
	"github.com/SplitFi/go-splitfi/util/retry"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func init() {
	env.RegisterValidation("GITHUB_API_KEY", "required")
}

const (
	pageSize = 100
	poolSize = 12
)

var ErrAPIKeyExpired = errors.New("coingecko api key expired")

var (
	baseURL, _                        = url.Parse("https://raw.github.com/trustwallet/assets/master")
	getTokensEndpointTemplate         = fmt.Sprintf("%s/%s", baseURL.String(), "blockchains/%s/tokenlist.json")
	getExtendedTokensEndpointTemplate = fmt.Sprintf("%s/%s", baseURL.String(), "blockchains/%s/tokenlist-extended.json")
)

// Map of chains to coingecko asset_platform identifiers
var chainToIdentifier = map[persist.Chain]string{
	persist.ChainETH:         "ethereum",
	persist.ChainPolygon:     "polygon",
	persist.ChainOptimism:    "optimism",
	persist.ChainArbitrum:    "arbitrum",
	persist.ChainBase:        "base",
	persist.ChainBaseSepolia: "", // no testnet support by trustwallet
}

type ErrGithubRateLimited struct{ Err error }

func (e ErrGithubRateLimited) Unwrap() error { return e.Err }
func (e ErrGithubRateLimited) Error() string {
	return fmt.Sprintf("rate limited by github for trustwallet: %s", e.Err)
}

// TWToken is a Token from trustwallet
type TWToken struct {
	Asset    string `json:"asset"`
	Type     string `json:"type"`
	Address  string `json:"address"`
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Decimals uint8  `json:"decimals"`
	LogoURI  string `json:"logoURI"`
}

type Provider struct {
	Chain persist.Chain
	r     *retry.Retryer
}

// NewProvider creates a new provider for OpenSea
func NewProvider(ctx context.Context, httpClient *http.Client, chain persist.Chain, l retry.Limiter) (*Provider, func()) {
	mustChainIdentifierFrom(chain)
	r, cleanup := retry.New(l, httpClient)
	return &Provider{Chain: chain, r: r}, cleanup
}

// GetTokensIncrementally returns a list of tokens
func (p *Provider) GetTokensIncrementally(ctx context.Context, address persist.Address) (<-chan []multichain.ChainAgnosticToken, <-chan error) {
	recCh := make(chan []multichain.ChainAgnosticToken, poolSize)
	errCh := make(chan error)
	assetsCh := make(chan twTokensReceived)
	go func() {
		defer close(assetsCh)
		streamTWTokensForAddress(ctx, p.r, p.Chain, address, assetsCh)
	}()
	go func() {
		defer close(recCh)
		defer close(errCh)
		p.streamTWTokensToTokens(ctx, assetsCh, recCh, errCh)
	}()
	return recCh, errCh
}

// GetTokens returns a list of tokens
func (p *Provider) GetTokens(ctx context.Context) ([]multichain.ChainAgnosticToken, error) {
	outCh := make(chan twTokensReceived)
	go func() {
		defer close(outCh)
		streamTWTokens(ctx, p.r, p.Chain, outCh)
	}()
	tokens, err := p.assetsToTokens(ctx, outCh)
	if err != nil {
		return nil, err
	}
	return tokens, nil
}

// GetTokenByAddress returns a single token for a contract address
func (p *Provider) GetTokenByAddress(ctx context.Context, address persist.Address) (multichain.ChainAgnosticToken, error) {
	outCh := make(chan twTokensReceived)
	go func() {
		defer close(outCh)
		streamTWTokensForAddress(ctx, p.r, p.Chain, address, outCh)
	}()
	tokens, err := p.assetsToTokens(ctx, outCh)
	if err != nil {
		return multichain.ChainAgnosticToken{}, err
	}

	var token multichain.ChainAgnosticToken
	if len(tokens) > 0 {
		token = tokens[0]
	}

	return token, nil
}

func (p *Provider) assetsToTokens(ctx context.Context, outCh <-chan twTokensReceived) (tokens []multichain.ChainAgnosticToken, err error) {
	recCh := make(chan []multichain.ChainAgnosticToken, poolSize)
	errCh := make(chan error)
	go func() {
		defer close(recCh)
		defer close(errCh)
		p.streamTWTokensToTokens(ctx, outCh, recCh, errCh)
	}()

	if err = <-errCh; err != nil {
		return nil, err
	}

	for page := range recCh {
		tokens = append(tokens, page...)
	}

	return tokens, nil
}

func (p *Provider) streamTWTokensToTokens(
	ctx context.Context,
	outCh <-chan twTokensReceived,
	recCh chan<- []multichain.ChainAgnosticToken,
	errCh chan<- error,
) {
	for page := range outCh {
		page := page

		if page.Err != nil {
			errCh <- wrapMissingContractErr(p.Chain, page.Err)
			return
		}

		if len(page.TWTokens) == 0 {
			continue
		}

		var out []multichain.ChainAgnosticToken

		// fetch tokens
		for _, asset := range page.TWTokens {
			token, err := p.assetToChainAgnosticToken(asset)
			if err != nil {
				errCh <- err
				return
			}

			out = append(out, *token)
		}

		recCh <- out
	}
}

func (p *Provider) assetToChainAgnosticToken(asset TWToken) (*multichain.ChainAgnosticToken, error) {
	typ, err := tokenTypeFromTWToken(asset)
	if err != nil {
		return nil, err
	}

	token := twTokenToAgnosticToken(asset, typ)
	return &token, nil
}

func checkURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

func mustTokensEndpoint(chain persist.Chain) *url.URL {
	s := fmt.Sprintf(getTokensEndpointTemplate, mustChainIdentifierFrom(chain))
	return checkURL(s)
}

func streamTWTokens(ctx context.Context, r *retry.Retryer, chain persist.Chain, outCh chan twTokensReceived) {
	endpoint := mustTokensEndpoint(chain)
	fetchTWTokens(ctx, r, mustAuthRequest(ctx, endpoint), outCh)
}

func streamTWTokensForAddress(ctx context.Context, r *retry.Retryer, chain persist.Chain, address persist.Address, outCh chan twTokensReceived) {
	endpoint := mustTokensEndpoint(chain)
	// setPagingParams(endpoint)
	fetchTWTokensFilter(ctx, r, mustAuthRequest(ctx, endpoint), outCh, func(a TWToken) bool {
		// trustwallet tokenlist doesn't let you filter, so we have to filter for only the token
		return persist.Address(a.Address) == address
	})

}

func tokenTypeFromTWToken(asset TWToken) (persist.TokenType, error) {
	switch asset.Type {
	case "erc20", "POLYGON":
		return persist.TokenTypeERC20, nil
	case "NATIVE":
		return persist.TokenTypeNative, nil
	default:
		return "", fmt.Errorf("unknown token type: %s", asset.Type)
	}
}

func twTokenToAgnosticToken(asset TWToken, tokenType persist.TokenType) multichain.ChainAgnosticToken {
	return multichain.ChainAgnosticToken{
		Address:   persist.Address(asset.Address),
		Symbol:    asset.Symbol,
		Name:      asset.Name,
		TokenType: tokenType,
		Logo:      persist.TokenLogo(asset.LogoURI),
		Decimals:  asset.Decimals,
		IsSpam:    util.ToPointer(contractNameIsSpam(asset.Name)),
	}
}

// mustAuthRequest returns a http.Request with authorization headers
func mustAuthRequest(ctx context.Context, url *url.URL) *http.Request {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("X-API-KEY", env.GetString("GITHUB_API_KEY"))
	return req
}

type tokenlistResult struct {
	Tokens []TWToken `json:"tokens"`
}

type errorResult struct {
	Errors []string `json:"errors"`
}

type twTokensReceived struct {
	TWTokens []TWToken
	Err      error
}

func wrapMissingContractErr(chain persist.Chain, err error) error {
	errMsg := err.Error()
	if strings.HasPrefix(errMsg, "Contract") && strings.HasSuffix(errMsg, "not found") {
		a := strings.TrimPrefix(errMsg, "Contract ")
		a = strings.TrimSuffix(a, " not found")
		a = strings.TrimSpace(a)
		return multichain.ErrProviderContractNotFound{
			Chain:    chain,
			Contract: persist.Address(a),
			Err:      err,
		}
	}
	return err
}

func readErrBody(ctx context.Context, body io.Reader) error {
	b := new(bytes.Buffer)

	_, err := b.ReadFrom(body)
	if err != nil {
		return err
	}

	byt := b.Bytes()

	var errResp errorResult

	err = json.Unmarshal(byt, &errResp)
	if err != nil {
		return err
	}

	if len(errResp.Errors) > 0 {
		err = fmt.Errorf(errResp.Errors[0])
		return err
	}

	return fmt.Errorf(string(byt))
}

func fetchTWTokens(ctx context.Context, r *retry.Retryer, req *http.Request, outCh chan twTokensReceived) {
	fetchTWTokensFilter(ctx, r, req, outCh, nil)
}

// fetchTWTokensFilter fetches assets from Trust wallet repo and sends them to outCh. An optional keepTWTokenFilter can be provided to filter out an asset
// after it is fetched if keepTWTokenFilter evaluates to false. This is useful for filtering out assets that can't be filtered natively by the API.
func fetchTWTokensFilter(ctx context.Context, r *retry.Retryer, req *http.Request, outCh chan twTokensReceived, keepTWTokenFilter func(a TWToken) bool) {
	for {
		resp, err := r.Do(req)
		if err != nil {
			err = wrapRateLimitErr(ctx, err)
			logger.For(ctx).Errorf("failed to get tokens from trustwallet: %s", err)
			outCh <- twTokensReceived{Err: err}
			return
		}

		defer resp.Body.Close()

		if resp.StatusCode == http.StatusUnauthorized {
			logger.For(ctx).Errorf("failed to get tokens from trustwallet: %s", ErrAPIKeyExpired)
			outCh <- twTokensReceived{Err: ErrAPIKeyExpired}
			return
		}

		if resp.StatusCode >= http.StatusInternalServerError {
			logger.For(ctx).Errorf("internal server error from trustwallet: %s", util.BodyAsError(resp))
			outCh <- twTokensReceived{Err: util.ErrHTTP{
				URL:    req.URL.String(),
				Status: resp.StatusCode,
				Err:    util.BodyAsError(resp),
			}}
		}

		if resp.StatusCode >= http.StatusBadRequest {
			err = readErrBody(ctx, resp.Body)
			logger.For(ctx).Errorf("unexpected status code (%d) from trustwallet: %s", resp.StatusCode, err)
			outCh <- twTokensReceived{Err: err}
			return
		}

		list := tokenlistResult{}

		if err := util.UnmarshallBody(&list, resp.Body); err != nil {
			logger.For(ctx).Errorf("failed to read response from trustwallet: %s", err)
			outCh <- twTokensReceived{Err: err}
			return
		}

		if keepTWTokenFilter == nil {
			logger.For(ctx).Infof("got %d tokens from trustwallet", len(list.Tokens))
			outCh <- twTokensReceived{TWTokens: list.Tokens}
		} else {
			filtered := util.Filter(list.Tokens, keepTWTokenFilter, true)
			logger.For(ctx).Infof("got %d tokens after filtering from trustwallet", len(list.Tokens))
			outCh <- twTokensReceived{TWTokens: filtered}
		}
		return
	}
}

func contractNameIsSpam(name string) bool {
	return strings.HasSuffix(strings.ToLower(name), ".lens-follower")
}

func mustChainIdentifierFrom(c persist.Chain) string {
	id, ok := chainToIdentifier[c]
	if !ok {
		panic(fmt.Sprintf("unknown chain identifier: %d", c))
	}
	return id
}

func wrapRateLimitErr(ctx context.Context, err error) error {
	if !errors.Is(err, retry.ErrOutOfRetries) {
		return err
	}
	err = ErrGithubRateLimited{err}
	logger.For(ctx).Error(err)
	sentryutil.ReportError(ctx, err)
	return err
}
