package persist

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"

	"github.com/lib/pq"

	"github.com/SplitFi/go-splitfi/util"
	"github.com/ethereum/go-ethereum/common"
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
	// MediaTypeBase64BMP represents a base64 encoded bmp file
	MediaTypeBase64BMP MediaType = "base64bmp"
	// MediaTypeText represents plain text
	MediaTypeText MediaType = "text"
	// MediaTypeHTML represents html
	MediaTypeHTML MediaType = "html"
	// MediaTypeBase64Text represents a base64 encoded plain text
	MediaTypeBase64Text MediaType = "base64text"
	// MediaTypeAudio represents audio
	MediaTypeAudio MediaType = "audio"
	// MediaTypeJSON represents json metadata
	MediaTypeJSON MediaType = "json"
	// MediaTypeAnimation represents an animation (.glb)
	MediaTypeAnimation MediaType = "animation"
	// MediaTypePDF represents a pdf
	MediaTypePDF MediaType = "pdf"
	// MediaTypeInvalid represents an invalid media type such as when a token's external metadata's API is broken or no longer exists
	MediaTypeInvalid MediaType = "invalid"
	// MediaTypeUnknown represents an unknown media type
	MediaTypeUnknown MediaType = "unknown"
	// MediaTypeSyncing represents a syncing media
	MediaTypeSyncing MediaType = "syncing"
)

var mediaTypePriorities = []MediaType{MediaTypeHTML, MediaTypeAudio, MediaTypeAnimation, MediaTypeVideo, MediaTypeBase64BMP, MediaTypeGIF, MediaTypeSVG, MediaTypeImage, MediaTypeJSON, MediaTypeBase64Text, MediaTypeText, MediaTypeSyncing, MediaTypeUnknown, MediaTypeInvalid}

const (
	// ChainETH represents the Ethereum blockchain
	ChainETH Chain = iota
	// ChainArbitrum represents the Arbitrum blockchain
	ChainArbitrum
	// ChainPolygon represents the Polygon/Matic blockchain
	ChainPolygon
	// ChainOptimism represents the Optimism blockchain
	ChainOptimism
	// MaxChainValue is the highest valid chain value, and should always be updated to
	// point to the most recently added chain type.
	MaxChainValue = ChainOptimism
)

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
	// URITypeBase64JSON represents a base64 encoded JSON document
	URITypeBase64JSON URIType = "base64json"
	// URITypeJSON represents a JSON document
	URITypeJSON URIType = "json"
	// URITypeBase64SVG represents a base64 encoded SVG
	URITypeBase64SVG URIType = "base64svg"
	//URITypeBase64BMP represents a base64 encoded BMP
	URITypeBase64BMP URIType = "base64bmp"
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

// ZeroAddress is the all-zero Ethereum address
const ZeroAddress EthereumAddress = "0x0000000000000000000000000000000000000000"

var gltfFields = []string{"scene", "scenes", "nodes", "meshes", "accessors", "bufferViews", "buffers", "materials", "textures", "images", "samplers", "cameras", "skins", "animations", "extensions", "extras"}

// EthereumAddress represents an Ethereum address
type EthereumAddress string

// EthereumAddressList is a slice of Addresses, used to implement scanner/valuer interfaces
type EthereumAddressList []EthereumAddress

func (l EthereumAddressList) Value() (driver.Value, error) {
	return pq.Array(l).Value()
}

// Scan implements the Scanner interface for the AddressList type
func (l *EthereumAddressList) Scan(value interface{}) error {
	return pq.Array(l).Scan(value)
}

// BlockNumber represents an Ethereum block number
type BlockNumber uint64

// BlockRange represents an inclusive block range
type BlockRange [2]BlockNumber

// TokenType represents the contract specification of the token
type TokenType string

// MediaType represents the type of media that a token
type MediaType string

// URIType represents the type of a URI
type URIType string

// TokenCountType represents the query of a token count operation
type TokenCountType string

// Chain represents which blockchain a token is on
type Chain int

// Logo represents the URL for an ERC20 token logo
type Logo string

// TokenMetadata represents the JSON metadata for a token
type TokenMetadata map[string]interface{}

// HexString represents a hex number of any size
type HexString string

// EthereumAddressAtBlock is an address connected to a block number
type EthereumAddressAtBlock struct {
	Address EthereumAddress `json:"address"`
	Block   BlockNumber     `json:"block"`
}

// EthereumTokenIdentifiers represents a unique identifier for a token on the Ethereum Blockchain
type EthereumTokenIdentifiers string

// Token represents an ERC20 or native token
type Token struct {
	Version         NullInt32       `json:"version"` // schema version for this model
	ID              DBID            `json:"id" binding:"required"`
	CreationTime    CreationTime    `json:"created_at"`
	Deleted         NullBool        `json:"-"`
	LastUpdated     LastUpdatedTime `json:"last_updated"`
	TokenType       TokenType       `json:"token_type"`
	ContractAddress EthereumAddress `json:"contract_address"`
	Chain           Chain           `json:"chain"`
	Name            NullString      `json:"name"`
	Symbol          NullString      `json:"symbol"`
	Decimals        NullInt32       `json:"decimals"`
	Logo            Logo            `json:"logo"`
	BlockNumber     BlockNumber     `json:"block_number"`
	IsSpam          *bool           `json:"is_spam"`
}

type Dimensions struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

func (d Dimensions) Valid() bool {
	return d.Width > 0 && d.Height > 0
}

// Media represents a splits media content with processed images from metadata
type Media struct {
	ThumbnailURL   NullString `json:"thumbnail_url,omitempty"`
	LivePreviewURL NullString `json:"live_preview_url,omitempty"`
	MediaURL       NullString `json:"media_url,omitempty"`
	MediaType      MediaType  `json:"media_type"`
	Dimensions     Dimensions `json:"dimensions"`
}

// IsServable returns true if the token's Media has enough information to serve it's assets.
func (m Media) IsServable() bool {
	return m.MediaURL != "" && m.MediaType.IsValid()
}

// TokenContract represents a smart contract's information for an ERC20 or native token
type TokenContract struct {
	ContractAddress     EthereumAddress `json:"address"`
	ContractName        NullString      `json:"name"`
	ContractImage       NullString      `json:"logo"`
	ContractSymbol      NullString      `json:"symbol"`
	ContractTotalSupply NullString      `json:"total_supply"`
}

// ContractCollectionNFT represents a contract within a collection nft
type ContractCollectionNFT struct {
	ContractName  NullString `json:"name"`
	ContractImage NullString `json:"image_url"`
}

type TokenUpdateTotalSupplyInput struct {
	TotalSupply HexString   `json:"total_supply"`
	BlockNumber BlockNumber `json:"block_number"`
}

// TokenRepository represents a repository for interacting with persisted tokens
type TokenRepository interface {
	GetByWallet(context.Context, EthereumAddress, int64, int64) ([]Token, error)
	GetByTokenIdentifiers(context.Context, EthereumAddress, int64, int64) ([]Token, error)
	GetByIdentifiers(context.Context, EthereumAddress) (Token, error)
	TokenExistsByTokenIdentifiers(context.Context, EthereumAddress) (bool, error)
	Upsert(context.Context, Token) error
	BulkUpsert(context.Context, []Token) error
	UpdateByID(context.Context, DBID, interface{}) error
	MostRecentBlock(context.Context) (BlockNumber, error)
	DeleteByID(context.Context, DBID) error
}

// ErrTokenNotFoundByTokenIdentifiers is an error that is returned when a token is not found by its identifiers (token ID and contract address)
type ErrTokenNotFoundByTokenIdentifiers struct {
	ContractAddress EthereumAddress
}

// ErrTokenNotFoundByIdentifiers is an error that is returned when a token is not found by its identifiers (token ID and contract address and owner address)
type ErrTokenNotFoundByIdentifiers struct {
	ContractAddress EthereumAddress
	OwnerAddress    EthereumAddress
}

// ErrTokenNotFoundByID is an error that is returned when a token is not found by its ID
type ErrTokenNotFoundByID struct {
	ID DBID
}

type ErrTokensNotFoundByContract struct {
	ContractAddress EthereumAddress
}

type svgXML struct {
	XMLName xml.Name `xml:"svg"`
}

// SniffMediaType will attempt to detect the media type for a given array of bytes
func SniffMediaType(buf []byte) (MediaType, string) {

	var asXML svgXML
	if err := xml.Unmarshal(buf, &asXML); err == nil {
		return MediaTypeSVG, "image/svg+xml"
	}

	contentType := http.DetectContentType(buf)
	contentType = strings.TrimSpace(contentType)
	whereCharset := strings.IndexByte(contentType, ';')
	if whereCharset != -1 {
		contentType = contentType[:whereCharset]
	}
	if contentType == "application/octet-stream" || contentType == "text/plain" {
		// fallback of http.DetectContentType
		if strings.EqualFold(string(buf[:4]), "glTF") {
			return MediaTypeAnimation, "model/gltf+binary"
		}

		if strings.HasPrefix(strings.TrimSpace(string(buf[:20])), "{") && util.ContainsAnyString(strings.TrimSpace(string(buf)), gltfFields...) {
			return MediaTypeAnimation, "model/gltf+json"
		}
	}
	return MediaFromContentType(contentType), contentType
}

// MediaFromContentType will attempt to convert a content type to a media type
func MediaFromContentType(contentType string) MediaType {
	contentType = strings.TrimSpace(contentType)
	whereCharset := strings.IndexByte(contentType, ';')
	if whereCharset != -1 {
		contentType = contentType[:whereCharset]
	}
	spl := strings.Split(contentType, "/")

	switch spl[0] {
	case "image":
		switch spl[1] {
		case "svg", "svg+xml":
			return MediaTypeSVG
		case "gif":
			return MediaTypeGIF
		default:
			return MediaTypeImage
		}
	case "video":
		return MediaTypeVideo
	case "audio":
		return MediaTypeAudio
	case "text":
		switch spl[1] {
		case "html":
			return MediaTypeHTML
		default:
			return MediaTypeText
		}
	case "application":
		switch spl[1] {
		case "pdf":
			return MediaTypePDF
		}
		fallthrough
	default:
		return MediaTypeUnknown
	}
}

func (e ErrTokenNotFoundByID) Error() string {
	return fmt.Sprintf("token not found by ID: %s", e.ID)
}

func (e ErrTokensNotFoundByContract) Error() string {
	return fmt.Sprintf("tokens not found by contract: %s", e.ContractAddress)
}

func (e ErrTokenNotFoundByTokenIdentifiers) Error() string {
	return fmt.Sprintf("token not found with contract address %s", e.ContractAddress)
}

func (e ErrTokenNotFoundByIdentifiers) Error() string {
	return fmt.Sprintf("token not found with contract address %s", e.ContractAddress)
}

// NormalizeAddress normalizes an address for the given chain
func (c Chain) NormalizeAddress(addr Address) string {
	switch c {
	case ChainETH:
		return strings.ToLower(addr.String())
	default:
		return addr.String()
	}
}

// BaseKeywords are the keywords that are default for discovering media for a given chain
func (c Chain) BaseKeywords() (image []string, anim []string) {
	return []string{"image"}, []string{"animation", "video"}
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
		return fmt.Errorf("Chain must be an string")
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
	}
}

// URL turns a token's URI into a URL
func (uri URIType) URL() (*url.URL, error) {
	return url.Parse(uri.String())
}

// IsPathPrefixed returns whether the URI is prefixed with a path to be parsed by a browser or decentralized storage service
func (uri URIType) IsPathPrefixed() bool {
	return strings.HasPrefix(uri.String(), "http") || strings.HasPrefix(uri.String(), "ipfs://") || strings.HasPrefix(uri.String(), "arweave") || strings.HasPrefix(uri.String(), "ar://")
}

func (uri URIType) String() string {
	asString := string(uri)
	if strings.HasPrefix(asString, "http") || strings.HasPrefix(asString, "ipfs") || strings.HasPrefix(asString, "ar") {
		url, err := url.QueryUnescape(string(uri))
		if err == nil && url != string(uri) {
			return url
		}
	}
	return asString
}

// Value implements the driver.Valuer interface for token URIs
func (uri URIType) Value() (driver.Value, error) {
	result := string(uri)
	if strings.Contains(result, "://") {
		result = url.QueryEscape(result)
	}
	clean := strings.Map(cleanString, result)
	return strings.ToValidUTF8(strings.ReplaceAll(clean, "\\u0000", ""), ""), nil
}

// Scan implements the sql.Scanner interface for token URIs
func (uri *URIType) Scan(src interface{}) error {
	if src == nil {
		*uri = URIType("")
		return nil
	}
	*uri = URIType(src.(string))
	return nil
}

// Type returns the type of the URI
func (uri URIType) Type() URIType {
	asString := uri.String()
	asString = strings.TrimSpace(asString)
	switch {
	case strings.HasPrefix(asString, "ipfs"), strings.HasPrefix(asString, "Qm"):
		return URITypeIPFS
	case strings.HasPrefix(asString, "ar://"), strings.HasPrefix(asString, "arweave://"):
		return URITypeArweave
	case strings.HasPrefix(asString, "data:application/json;base64,"):
		return URITypeBase64JSON
	case strings.HasPrefix(asString, "data:image/svg+xml;base64,"), strings.HasPrefix(asString, "data:image/svg xml;base64,"):
		return URITypeBase64SVG
	case strings.HasPrefix(asString, "data:image/bmp;base64,"):
		return URITypeBase64BMP
	case strings.Contains(asString, "ipfs.io/api"):
		return URITypeIPFSAPI
	case strings.Contains(asString, "/ipfs/"):
		return URITypeIPFSGateway
	case strings.HasPrefix(asString, "http"), strings.HasPrefix(asString, "https"):
		return URITypeHTTP
	case strings.HasPrefix(asString, "{"), strings.HasPrefix(asString, "["), strings.HasPrefix(asString, "data:application/json"), strings.HasPrefix(asString, "data:text/plain,{"):
		return URITypeJSON
	case strings.HasPrefix(asString, "<svg"), strings.HasPrefix(asString, "data:image/svg+xml;utf8,"), strings.HasPrefix(asString, "data:image/svg+xml,"), strings.HasPrefix(asString, "data:image/svg xml,"):
		return URITypeSVG
	case strings.HasSuffix(asString, ".ens"):
		return URITypeENS
	case asString == URITypeInvalid.String():
		return URITypeInvalid
	case asString == "":
		return URITypeNone
	default:
		return URITypeUnknown
	}
}

// IsRenderable returns whether a frontend could render the given URI directly
func (uri URIType) IsRenderable() bool {
	return uri.IsHTTP() // || uri.IsIPFS() || uri.IsArweave()
}

// IsHTTP returns whether a frontend could render the given URI directly
func (uri URIType) IsHTTP() bool {
	asString := uri.String()
	asString = strings.TrimSpace(asString)
	return strings.HasPrefix(asString, "http")
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

// BigInt returns the hex string as a big.Int
func (hex HexString) BigInt() *big.Int {
	it, ok := big.NewInt(0).SetString(hex.String(), 16)
	if !ok {
		it, ok = big.NewInt(0).SetString(hex.String(), 10)
		if !ok {
			return big.NewInt(0)
		}
	}
	return it
}

// Add adds the given hex string to the current hex string
func (hex HexString) Add(new HexString) HexString {
	asInt := hex.BigInt()
	return HexString(asInt.Add(asInt, new.BigInt()).Text(16))
}

// Value implements the driver.Valuer interface for media
func (m Media) Value() (driver.Value, error) {
	return json.Marshal(m)
}

// Scan implements the sql.Scanner interface for media
func (m *Media) Scan(src interface{}) error {
	if src == nil {
		*m = Media{}
		return nil
	}
	return json.Unmarshal(src.([]uint8), &m)
}

func (a EthereumAddress) String() string {
	return normalizeAddress(strings.ToLower(string(a)))
}

// Address returns the ethereum address byte array
func (a EthereumAddress) Address() common.Address {
	return common.HexToAddress(a.String())
}

// Value implements the database/sql/driver Valuer interface for the address type
func (a EthereumAddress) Value() (driver.Value, error) {
	return a.String(), nil
}

// MarshallJSON implements the json.Marshaller interface for the address type
func (a EthereumAddress) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

// UnmarshalJSON implements the json.Unmarshaller interface for the address type
func (a *EthereumAddress) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	*a = EthereumAddress(normalizeAddress(strings.ToLower(s)))
	return nil
}

// Scan implements the database/sql Scanner interface
func (a *EthereumAddress) Scan(i interface{}) error {
	if i == nil {
		*a = EthereumAddress("")
		return nil
	}
	if it, ok := i.(string); ok {
		*a = EthereumAddress(it)
		return nil
	}
	*a = EthereumAddress(i.([]uint8))
	return nil
}

// Uint64 returns the ethereum block number as a uint64
func (b BlockNumber) Uint64() uint64 {
	return uint64(b)
}

// BigInt returns the ethereum block number as a big.Int
func (b BlockNumber) BigInt() *big.Int {
	return new(big.Int).SetUint64(b.Uint64())
}

func (b BlockNumber) String() string {
	return strings.ToLower(b.BigInt().String())
}

// Hex returns the ethereum block number as a hex string
func (b BlockNumber) Hex() string {
	return strings.ToLower(b.BigInt().Text(16))
}

// Value implements the database/sql/driver Valuer interface for the block number type
func (b BlockNumber) Value() (driver.Value, error) {
	return b.BigInt().Int64(), nil
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
	// Replace literal '\\u0000' with empty string (marshal to JSON escapes each backslash)
	cleaned = strings.ReplaceAll(cleaned, "\\\\u0000", "")
	// Replace unicode NULL char (u+0000) i.e. '\u0000' with empty string
	cleaned = strings.ReplaceAll(cleaned, "\\u0000", "")
	return []byte(cleaned), nil
}

// Scan implements the database/sql Scanner interface for the AddressAtBlock type
func (a *EthereumAddressAtBlock) Scan(src interface{}) error {
	if src == nil {
		*a = EthereumAddressAtBlock{}
		return nil
	}
	return json.Unmarshal(src.([]uint8), a)
}

// Value implements the database/sql/driver Valuer interface for the AddressAtBlock type
func (a EthereumAddressAtBlock) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// IsValid returns true if the media type is not unknown, syncing, or invalid
func (m MediaType) IsValid() bool {
	return m != MediaTypeUnknown && m != MediaTypeInvalid && m != MediaTypeSyncing && m != ""
}

// IsImageLike returns true if the media type is a type that is expected to be like an image and not live render
func (m MediaType) IsImageLike() bool {
	return m == MediaTypeImage || m == MediaTypeGIF || m == MediaTypeBase64BMP || m == MediaTypeSVG
}

// IsAnimationLike returns true if the media type is a type that is expected to be like an animation and live render
func (m MediaType) IsAnimationLike() bool {
	return m == MediaTypeVideo || m == MediaTypeHTML || m == MediaTypeAudio || m == MediaTypeAnimation
}

// IsMorePriorityThan returns true if the media type is more important than the other media type
func (m MediaType) IsMorePriorityThan(other MediaType) bool {
	for _, t := range mediaTypePriorities {
		if t == m {
			return true
		}
		if t == other {
			return false
		}
	}
	return true
}

// Value implements the database/sql/driver Valuer interface for the MediaType type
func (m MediaType) Value() (driver.Value, error) {
	return string(m), nil
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

// NewEthereumTokenIdentifiers creates a new token identifiers
func NewEthereumTokenIdentifiers(pContractAddress EthereumAddress) EthereumTokenIdentifiers {
	return EthereumTokenIdentifiers(fmt.Sprintf("%s", pContractAddress))
}

func (t EthereumTokenIdentifiers) String() string {
	return string(t)
}

// GetParts returns the parts of the token identifiers
func (t EthereumTokenIdentifiers) GetParts() (EthereumAddress, error) {
	parts := strings.Split(t.String(), "+")
	if len(parts) != 1 {
		return "", fmt.Errorf("invalid token identifiers: %s", t)
	}
	return EthereumAddress(EthereumAddress(parts[0]).String()), nil
}

// Value implements the driver.Valuer interface
func (t EthereumTokenIdentifiers) Value() (driver.Value, error) {
	return t.String(), nil
}

// Scan implements the database/sql Scanner interface for the TokenIdentifiers type
func (t *EthereumTokenIdentifiers) Scan(i interface{}) error {
	if i == nil {
		*t = ""
		return nil
	}
	res := strings.Split(i.(string), "+")
	if len(res) != 1 {
		return fmt.Errorf("invalid token identifiers: %v", i)
	}
	*t = EthereumTokenIdentifiers(fmt.Sprintf("%s", res[0]))
	return nil
}

func normalizeAddress(address string) string {
	withoutPrefix := strings.TrimPrefix(address, "0x")
	if len(withoutPrefix) < 40 {
		return ""
	}
	return "0x" + withoutPrefix[len(withoutPrefix)-40:]
}

func WalletsToEthereumAddresses(pWallets []Wallet) []EthereumAddress {
	result := make([]EthereumAddress, len(pWallets))
	for i, wallet := range pWallets {
		result[i] = EthereumAddress(wallet.Address)
	}
	return result
}
