package persist

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/mutuals/go-mutuals/util"
)

const (
	// TokenTypeERC20 is the type of ERC20 token
	TokenTypeERC20 TokenType = "ERC-20"
	// TokenTypeNative is the type of a native token
	TokenTypeNative TokenType = "NATIVE"
)

const (
	// MediaTypeVideo represents a video
	MediaTypeVideo MediaType = "video"
	// MediaTypeImage represents an image
	MediaTypeImage MediaType = "image"
	// MediaTypeGIF represents a gif
	MediaTypeGIF MediaType = "gif"
	// MediaTypeSVG represents an SVG
	MediaTypeSVG MediaType = "svg"
	// MediaTypeText represents plain text
	MediaTypeText MediaType = "text"
	// MediaTypeHTML represents html
	MediaTypeHTML MediaType = "html"
	// MediaTypeAudio represents audio
	MediaTypeAudio MediaType = "audio"
	// MediaTypeJSON represents json metadata
	MediaTypeJSON MediaType = "json"
	// MediaTypeAnimation represents an animation (.glb)
	MediaTypeAnimation MediaType = "animation"
	// MediaTypePDF represents a pdf
	MediaTypePDF MediaType = "pdf"
	// MediaTypeInvalid represents an invalid media type
	MediaTypeInvalid MediaType = "invalid"
	// MediaTypeUnknown represents an unknown media type
	MediaTypeUnknown MediaType = "unknown"
	// MediaTypeSyncing represents a syncing media
	MediaTypeSyncing MediaType = "syncing"
	// MediaTypeFallback represents a fallback media
	MediaTypeFallback MediaType = "fallback"
)

func (m MediaType) ToContentType() string {
	switch m {
	case MediaTypeVideo:
		return "video/mp4"
	case MediaTypeImage:
		return "image/jpeg"
	case MediaTypeGIF:
		return "image/gif"
	case MediaTypeSVG:
		return "image/svg+xml"
	case MediaTypeText:
		return "text/plain"
	case MediaTypeHTML:
		return "text/html"
	case MediaTypeAudio:
		return "audio/mpeg"
	case MediaTypeJSON:
		return "application/json"
	case MediaTypeAnimation:
		return "model/gltf-binary"
	case MediaTypePDF:
		return "application/pdf"
	default:
		return ""
	}
}

const (
	// ChainETH represents the Ethereum blockchain
	ChainETH Chain = iota
	// ChainArbitrum represents the Arbitrum blockchain
	ChainArbitrum
	// ChainPolygon represents the Polygon/Matic blockchain
	ChainPolygon
	// ChainOptimism represents the Optimism blockchain
	ChainOptimism
	// ChainBase represents the base chain
	ChainBase
	// ChainSepolia - Ethereum testnet
	ChainSepolia
	// ChainBaseSepolia - Base testnet
	ChainBaseSepolia

	// MaxChainValue is the highest valid chain value
	MaxChainValue = ChainBaseSepolia
)

var L1Chains = map[Chain]L1Chain{
	ChainOptimism:    L1Chain(ChainETH),
	ChainPolygon:     L1Chain(ChainETH),
	ChainArbitrum:    L1Chain(ChainETH),
	ChainBase:        L1Chain(ChainETH),
	ChainBaseSepolia: L1Chain(ChainETH),
	ChainETH:         L1Chain(ChainETH),
}

var L1ChainGroups = map[L1Chain][]Chain{
	L1Chain(ChainETH): EvmChains,
}

var AllChains = []Chain{ChainETH, ChainArbitrum, ChainPolygon, ChainOptimism, ChainBase}
var EvmChains = util.MapKeys(evmChains)
var evmChains map[Chain]bool = map[Chain]bool{
	ChainETH:      true,
	ChainOptimism: true,
	ChainPolygon:  true,
	ChainArbitrum: true,
	ChainBase:     true,
}

const (
	// URITypeIPFS represents an IPFS URI
	URITypeIPFS URIType = "ipfs"
	// URITypeArweave represents an Arweave URI
	URITypeArweave URIType = "arweave"
	// URITypeHTTP represents an HTTP URI
	URITypeHTTP URIType = "http"
	// URITypeIPFSAPI represents an IPFS API URI
	URITypeIPFSAPI URIType = "ipfs-api"
	// URITypeIPFSGateway represents an IPFS Gateway URI
	URITypeIPFSGateway URIType = "ipfs-gateway"
	// URITypeArweaveGateway represents an Arweave Gateway URI
	URITypeArweaveGateway URIType = "arweave-gateway"
	// URITypeBase64JSON represents a base64 encoded JSON document
	URITypeBase64JSON URIType = "base64json"
	// URITypeBase64HTML represents a base64 encoded HTML document
	URITypeBase64HTML URIType = "base64html"
	// URITypeJSON represents a JSON document
	URITypeJSON URIType = "json"
	// URITypeBase64SVG represents a base64 encoded SVG
	URITypeBase64SVG URIType = "base64svg"
	// URITypeBase64BMP represents a base64 encoded BMP
	URITypeBase64BMP URIType = "base64bmp"
	// URITypeBase64PNG represents a base64 encoded PNG
	URITypeBase64PNG URIType = "base64png"
	// URITypeBase64JPEG represents a base64 encoded JPEG
	URITypeBase64JPEG URIType = "base64jpeg"
	// URITypeBase64GIF represents a base64 encoded GIF
	URITypeBase64GIF URIType = "base64gif"
	// URITypeBase64WAV represents a base64 encoded WAV
	URITypeBase64WAV URIType = "base64wav"
	// URITypeBase64MP3 represents a base64 encoded MP3
	URITypeBase64MP3 URIType = "base64mp3"
	// URITypeSVG represents an SVG
	URITypeSVG URIType = "svg"
	// URITypeENS represents an ENS domain
	URITypeENS URIType = "ens"
	// URITypeUnknown represents an unknown URI type
	URITypeUnknown URIType = "unknown"
	// URITypeInvalid represents an invalid URI type
	URITypeInvalid URIType = "invalid"
	// URITypeNone represents no URI
	URITypeNone URIType = "none"
)

func (u URIType) IsRaw() bool {
	switch u {
	case URITypeBase64JSON, URITypeBase64HTML, URITypeBase64SVG, URITypeBase64BMP, URITypeBase64PNG, URITypeBase64JPEG, URITypeBase64GIF, URITypeBase64WAV, URITypeBase64MP3, URITypeJSON, URITypeSVG, URITypeENS:
		return true
	default:
		return false
	}
}

func (u URIType) ToMediaType() MediaType {
	switch u {
	case URITypeBase64JSON, URITypeJSON:
		return MediaTypeJSON
	case URITypeBase64SVG, URITypeSVG:
		return MediaTypeSVG
	case URITypeBase64BMP:
		return MediaTypeImage
	case URITypeBase64PNG:
		return MediaTypeImage
	case URITypeBase64HTML:
		return MediaTypeHTML
	case URITypeBase64JPEG:
		return MediaTypeImage
	case URITypeBase64GIF:
		return MediaTypeGIF
	case URITypeBase64MP3:
		return MediaTypeAudio
	case URITypeBase64WAV:
		return MediaTypeAudio
	default:
		return MediaTypeUnknown
	}
}

// InvalidTokenURI represents an invalid token URI
const InvalidTokenURI TokenURI = "INVALID"

// EthereumAddress represents an Ethereum address
type EthereumAddress string

// BlockNumber represents an Ethereum block number
type BlockNumber uint64

// TokenType represents the contract specification of the token
type TokenType string

// MediaType represents the type of media that a token has
type MediaType string

// URIType represents the type of a URI
type URIType string

// Chain represents which blockchain a token is on
type Chain int

type L1Chain Chain

// TokenURI represents the URI for an Ethereum token
type TokenURI string

// TokenMetadata represents the JSON metadata for a token
type TokenMetadata map[string]interface{}

// HexString represents a hex number of any size
type HexString string

// TokenChainAddress represents an address and a chain for a token
type TokenChainAddress struct {
	Address Address
	Chain   Chain
}

// NewTokenChainAddress creates a new token chain address
func NewTokenChainAddress(pContractAddress Address, pChain Chain) TokenChainAddress {
	return TokenChainAddress{
		Address: pContractAddress,
		Chain:   pChain,
	}
}

func (t TokenChainAddress) String() string {
	return fmt.Sprintf("%s+%d", t.Chain.NormalizeAddress(t.Address), t.Chain)
}

// Value implements the driver.Valuer interface
func (t TokenChainAddress) Value() (driver.Value, error) {
	return t.String(), nil
}

// Scan implements the database/sql Scanner interface for the TokenChainAddress type
func (t *TokenChainAddress) Scan(i interface{}) error {
	if i == nil {
		*t = TokenChainAddress{}
		return nil
	}
	res := strings.Split(i.(string), "+")
	if len(res) != 2 {
		return fmt.Errorf("invalid token chain address: %v - %T", i, i)
	}
	chain, err := strconv.Atoi(res[1])
	if err != nil {
		return err
	}
	*t = TokenChainAddress{
		Address: Address(res[0]),
		Chain:   Chain(chain),
	}
	return nil
}

func (t TokenChainAddress) ToL1ChainAddress() L1ChainAddress {
	return NewL1ChainAddress(t.Address, t.Chain)
}

var errTokenNotFound ErrTokenNotFound

type ErrTokenNotFound struct{}

func (e ErrTokenNotFound) Unwrap() error { return notFoundError }
func (e ErrTokenNotFound) Error() string { return "token not found" }

// ErrTokenNotFoundByID is returned when a token is not found by its ID
type ErrTokenNotFoundByID struct {
	ID DBID
}

func (e ErrTokenNotFoundByID) Unwrap() error { return errTokenNotFound }
func (e ErrTokenNotFoundByID) Error() string {
	return fmt.Sprintf("token not found by ID: %s", e.ID)
}

// NormalizeAddress normalizes an address for the given chain
func (c Chain) NormalizeAddress(addr Address) string {
	if evmChains[c] {
		return strings.ToLower(addr.String())
	}
	return addr.String()
}


// Value implements the driver.Valuer interface for the Chain type
func (c Chain) Value() (driver.Value, error) {
	return c, nil
}

// Scan implements the sql.Scanner interface for the Chain type
func (c *Chain) Scan(src interface{}) error {
	if src == nil {
		*c = Chain(0)
		return nil
	}
	*c = Chain(src.(int64))
	return nil
}

// UnmarshalJSON will unmarshall the JSON data into the Chain type
func (c *Chain) UnmarshalJSON(data []byte) error {
	var s int
	var asString string
	if err := json.Unmarshal(data, &s); err != nil {
		err = json.Unmarshal(data, &asString)
		if err != nil {
			return err
		}
		switch strings.ToLower(asString) {
		case "ethereum":
			*c = ChainETH
		case "arbitrum":
			*c = ChainArbitrum
		case "polygon":
			*c = ChainPolygon
		case "optimism":
			*c = ChainOptimism
		case "base":
			*c = ChainBase
		}
		return nil
	}
	*c = Chain(s)
	return nil
}

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (c *Chain) UnmarshalGQL(v interface{}) error {
	n, ok := v.(string)
	if !ok {
		return fmt.Errorf("Chain must be a string")
	}

	switch strings.ToLower(n) {
	case "ethereum":
		*c = ChainETH
	case "arbitrum":
		*c = ChainArbitrum
	case "polygon":
		*c = ChainPolygon
	case "optimism":
		*c = ChainOptimism
	case "base":
		*c = ChainBase
	}
	return nil
}

// MarshalGQL implements the graphql.Marshaler interface
func (c Chain) MarshalGQL(w io.Writer) {
	switch c {
	case ChainETH:
		w.Write([]byte(`"Ethereum"`))
	case ChainArbitrum:
		w.Write([]byte(`"Arbitrum"`))
	case ChainPolygon:
		w.Write([]byte(`"Polygon"`))
	case ChainOptimism:
		w.Write([]byte(`"Optimism"`))
	case ChainBase:
		w.Write([]byte(`"Base"`))
	}
}

func (c Chain) L1Chain() L1Chain {
	lc, ok := L1Chains[c]
	if !ok {
		panic("l1 chain not found")
	}
	return lc
}

func (c Chain) L1ChainGroup() []Chain {
	cg, ok := L1ChainGroups[c.L1Chain()]
	if !ok {
		panic("chain group not found")
	}
	return cg
}

func (uri TokenURI) String() string { return string(uri) }

// Value implements the driver.Valuer interface for token URIs
func (uri TokenURI) Value() (driver.Value, error) {
	result := string(uri)
	if strings.Contains(result, "://") {
		result = url.QueryEscape(result)
	}
	clean := strings.Map(cleanString, result)
	return strings.ToValidUTF8(strings.ReplaceAll(clean, "\\u0000", ""), ""), nil
}

// Scan implements the sql.Scanner interface for token URIs
func (uri *TokenURI) Scan(src interface{}) error {
	if src == nil {
		*uri = TokenURI("")
		return nil
	}
	*uri = TokenURI(src.(string))
	return nil
}

// Type returns the type of the token URI
func (uri TokenURI) Type() URIType {
	asString := uri.String()
	asString = strings.TrimSpace(asString)
	switch {
	case strings.HasPrefix(asString, "ipfs"), strings.HasPrefix(asString, "Qm"):
		return URITypeIPFS
	case strings.HasPrefix(asString, "ar://"), strings.HasPrefix(asString, "arweave://"):
		return URITypeArweave
	case strings.HasPrefix(asString, "data:text/html;base64,"), strings.HasPrefix(asString, "data:text/html;charset=utf-8;base64,"), strings.HasPrefix(asString, "data:text/html") && strings.Contains(asString, ";base64,"):
		return URITypeBase64HTML
	case strings.HasPrefix(asString, "data:application/json;base64,"), strings.HasPrefix(asString, "data:application/json;charset=utf-8;base64,"), strings.HasPrefix(asString, "data:application/json") && strings.Contains(asString, ";base64,"):
		return URITypeBase64JSON
	case strings.HasPrefix(asString, "data:image/svg+xml;base64,"), strings.HasPrefix(asString, "data:image/svg xml;base64,"), strings.HasPrefix(asString, "data:image/svg+xml") && strings.Contains(asString, ";base64,"), strings.HasPrefix(asString, "data:image/svg xml") && strings.Contains(asString, ";base64,"):
		return URITypeBase64SVG
	case strings.HasPrefix(asString, "data:image/bmp;base64,"), strings.HasPrefix(asString, "data:image/bmp;charset=utf-8;base64,"), strings.HasPrefix(asString, "data:image/bmp") && strings.Contains(asString, ";base64,"):
		return URITypeBase64BMP
	case strings.HasPrefix(asString, "data:image/png;base64,"), strings.HasPrefix(asString, "data:image/png;charset=utf-8;base64,"), strings.HasPrefix(asString, "data:image/png") && strings.Contains(asString, ";base64,"):
		return URITypeBase64PNG
	case strings.HasPrefix(asString, "data:image/jpeg;base64,"), strings.HasPrefix(asString, "data:image/jpeg;charset=utf-8;base64,"), strings.HasPrefix(asString, "data:image/jpeg") && strings.Contains(asString, ";base64,"):
		return URITypeBase64JPEG
	case strings.HasPrefix(asString, "data:image/gif;base64,"), strings.HasPrefix(asString, "data:image/gif;charset=utf-8;base64,"), strings.HasPrefix(asString, "data:image/gif") && strings.Contains(asString, ";base64,"):
		return URITypeBase64GIF
	case strings.HasPrefix(asString, "data:audio/wav;base64,"), strings.HasPrefix(asString, "data:audio/wav;charset=utf-8;base64,"), strings.HasPrefix(asString, "data:audio/wav") && strings.Contains(asString, ";base64,"):
		return URITypeBase64WAV
	case strings.HasPrefix(asString, "data:audio/mpeg;base64,"), strings.HasPrefix(asString, "data:audio/mpeg;charset=utf-8;base64,"), strings.HasPrefix(asString, "data:audio/mpeg") && strings.Contains(asString, ";base64,"):
		return URITypeBase64MP3
	case strings.Contains(asString, "ipfs.io/api"):
		return URITypeIPFSAPI
	case strings.Contains(asString, "/ipfs/"):
		return URITypeIPFSGateway
	case strings.HasPrefix(asString, "https://arweave.net/"):
		return URITypeArweaveGateway
	case strings.HasPrefix(asString, "http"), strings.HasPrefix(asString, "https"):
		return URITypeHTTP
	case strings.HasPrefix(asString, "{"), strings.HasPrefix(asString, "["), strings.HasPrefix(asString, "data:application/json"), strings.HasPrefix(asString, "data:text/plain,{"):
		return URITypeJSON
	case strings.HasPrefix(asString, "<svg"), strings.HasPrefix(asString, "data:image/svg+xml"), strings.HasPrefix(asString, "data:image/svg xml"):
		return URITypeSVG
	case strings.HasSuffix(asString, ".ens"):
		return URITypeENS
	case asString == InvalidTokenURI.String():
		return URITypeInvalid
	case asString == "":
		return URITypeNone
	default:
		return URITypeUnknown
	}
}

// Uint64 returns the ethereum block number as a uint64
func (b BlockNumber) Uint64() uint64 {
	return uint64(b)
}

func (b BlockNumber) String() string {
	return fmt.Sprintf("%d", b.Uint64())
}

// Hex returns the ethereum block number as a hex string
func (b BlockNumber) Hex() string {
	return fmt.Sprintf("%x", b.Uint64())
}

// Value implements the database/sql/driver Valuer interface for the block number type
func (b BlockNumber) Value() (driver.Value, error) {
	return int64(b.Uint64()), nil
}

// Scan implements the database/sql Scanner interface for the block number type
func (b *BlockNumber) Scan(src interface{}) error {
	if src == nil {
		*b = BlockNumber(0)
		return nil
	}
	*b = BlockNumber(src.(int64))
	return nil
}

// Scan implements the database/sql Scanner interface for the TokenMetadata type
func (m *TokenMetadata) Scan(src interface{}) error {
	if src == nil {
		*m = TokenMetadata{}
		return nil
	}
	return json.Unmarshal(src.([]uint8), m)
}

// Value implements the database/sql/driver Valuer interface for the TokenMetadata type
func (m TokenMetadata) Value() (driver.Value, error) {
	return m.MarshalJSON()
}

// MarshalJSON implements the json.Marshaller interface for the TokenMetadata type
func (m TokenMetadata) MarshalJSON() ([]byte, error) {
	asMap := map[string]interface{}(m)
	val, err := json.Marshal(asMap)
	if err != nil {
		return nil, err
	}
	cleaned := strings.ToValidUTF8(string(val), "")
	cleaned = strings.ReplaceAll(cleaned, "\\\\u0000", "")
	cleaned = strings.ReplaceAll(cleaned, "\\u0000", "")
	return []byte(cleaned), nil
}

// IsValid returns true if the media type is not unknown, syncing, or invalid
func (m MediaType) IsValid() bool {
	return m != MediaTypeUnknown && m != MediaTypeInvalid && m != MediaTypeSyncing && m != ""
}

// IsImageLike returns true if the media type is expected to be like an image
func (m MediaType) IsImageLike() bool {
	return m == MediaTypeImage || m == MediaTypeGIF || m == MediaTypeSVG
}

// IsAnimationLike returns true if the media type is expected to be like an animation
func (m MediaType) IsAnimationLike() bool {
	return m == MediaTypeVideo || m == MediaTypeHTML || m == MediaTypeAudio || m == MediaTypeAnimation
}

// Value implements the database/sql/driver Valuer interface for the MediaType type
func (m MediaType) Value() (driver.Value, error) {
	return m.String(), nil
}

func (m MediaType) String() string {
	return string(m)
}

// Scan implements the database/sql Scanner interface for the MediaType type
func (m *MediaType) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	*m = MediaType(src.(string))
	return nil
}

func (t TokenType) String() string {
	return string(t)
}

// Value implements the database/sql/driver Valuer interface for the TokenType type
func (t TokenType) Value() (driver.Value, error) {
	return t.String(), nil
}

// Scan implements the database/sql Scanner interface for the TokenType type
func (t *TokenType) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	*t = TokenType(src.(string))
	return nil
}

func (hex HexString) String() string {
	return strings.TrimPrefix(strings.ToLower(string(hex)), "0x")
}

// Value implements the driver.Valuer interface for hex strings
func (hex HexString) Value() (driver.Value, error) {
	return hex.String(), nil
}

// Scan implements the sql.Scanner interface for hex strings
func (hex *HexString) Scan(src interface{}) error {
	if src == nil {
		*hex = HexString("")
		return nil
	}
	*hex = HexString(src.(string))
	return nil
}



