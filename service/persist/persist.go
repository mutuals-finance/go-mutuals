package persist

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/jackc/pgtype"
	"github.com/lib/pq"
	"github.com/segmentio/ksuid"
)

var cleanString = func(r rune) rune {
	if unicode.IsGraphic(r) || unicode.IsPrint(r) {
		return r
	}
	return -1
}

// DBID represents a database ID (application-wide unique identifier).
type DBID string

// UnmarshalGQL implements the graphql.Unmarshaler interface.
// It parses a Global ID (e.g., "User:123") and extracts the database ID ("123")
func (d *DBID) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("IDs must be strings")
	}

	// Logic: Split "Type:ID" and keep the ID part.
	parts := strings.Split(str, ":")
	if len(parts) < 2 {
		return fmt.Errorf("invalid GraphQL ID format: %s", str)
	}

	// We assume the ID is the second part.
	// If the ID itself contains colons, we join the rest.
	*d = DBID(strings.Join(parts[1:], ":"))
	return nil
}

// MarshalGQL implements the graphql.Marshaler interface.
// It writes the raw DBID as a string to the response.
func (d DBID) MarshalGQL(w io.Writer) {
	// Note: GraphQL clients usually expect the Global ID (Type:ID).
	io.WriteString(w, fmt.Sprintf(`"%s"`, d))
}

// String returns the string representation of the DBID.
func (d DBID) String() string {
	return string(d)
}

// Value implements the database/sql driver Valuer interface.
func (d DBID) Value() (driver.Value, error) {
	return d.String(), nil
}

// Scan implements the database/sql Scanner interface.
func (d *DBID) Scan(i interface{}) error {
	if i == nil {
		*d = ""
		return nil
	}
	if it, ok := i.([]uint8); ok {
		*d = DBID(it)
		return nil
	}
	switch v := i.(type) {
	case DBID:
		*d = v
	case string:
		*d = DBID(v)
	}
	return nil
}

// DBIDList is a slice of DBIDs, primarily used to implement scanner/valuer interfaces for Postgres Arrays.
type DBIDList []DBID

// Value implements the database/sql driver Valuer interface for DBIDList.
func (l DBIDList) Value() (driver.Value, error) {
	return pq.Array(l).Value()
}

// Scan implements the database/sql Scanner interface for DBIDList.
func (l *DBIDList) Scan(value interface{}) error {
	return pq.Array(l).Scan(value)
}

type DBIDTuple [2]DBID

// ErrNotFound is a general error for when some entity is not found.
var notFoundError ErrNotFound

// ErrNotFound should be wrapped to provide more details (e.g. "User not found").
type ErrNotFound struct{}

func (e ErrNotFound) Error() string { return "entity not found" }

// JSON represents arbitrary JSON data
type JSON json.RawMessage

// Scan implements the database/sql Scanner interface for the JSON type
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = JSON([]byte("null"))
		return nil
	}

	switch v := value.(type) {
	case []byte:
		*j = JSON(v)
		return nil
	case string:
		*j = JSON(v)
		return nil
	case pgtype.JSONB:
		*j = JSON(v.Bytes)
		return nil
	default:
		return fmt.Errorf("cannot scan type %T into JSON", value)
	}
}

// Value implements the database/sql driver Valuer interface for the JSON type.
func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return []byte(j), nil
}

// UnmarshalGQL implements the graphql.Unmarshaler interface.
func (j *JSON) UnmarshalGQL(v interface{}) error {
	switch v := v.(type) {
	case string:
		*j = JSON(v)
		return nil
	case []byte:
		*j = JSON(v)
		return nil
	case json.RawMessage:
		*j = JSON(v)
		return nil
	case map[string]interface{}, []interface{}:
		bytes, err := json.Marshal(v)
		if err != nil {
			return err
		}
		*j = JSON(bytes)
		return nil
	default:
		return fmt.Errorf("unable to unmarshal JSON from type %T", v)
	}
}

// MarshalGQL implements the graphql.Marshaler interface.
func (j JSON) MarshalGQL(w io.Writer) {
	if len(j) == 0 {
		w.Write([]byte("null"))
		return
	}
	w.Write(j)
}

// NullString represents a string that may be null in the DB.
type NullString string

func (n NullString) String() string {
	return string(n)
}

// Value implements the database/sql driver Valuer interface.
func (n NullString) Value() (driver.Value, error) {
	return strings.ToValidUTF8(strings.ReplaceAll(n.String(), "\\u0000", ""), ""), nil
}

// Scan implements the database/sql Scanner interface.
func (n *NullString) Scan(value interface{}) error {
	if value == nil {
		*n = NullString("")
		return nil
	}
	*n = NullString(value.(string))
	return nil
}

// DBIDPtrToNullString converts a DBID pointer to NullString
func DBIDPtrToNullString(id *DBID) NullString {
	if id == nil || *id == "" {
		return NullString("")
	}
	return NullString(id.String())
}

// DBIDPtrToSQLNullString converts a DBID pointer to sql.NullString
func DBIDPtrToSQLNullString(id *DBID) sql.NullString {
	if id == nil || *id == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: id.String(), Valid: true}
}

// NullStringToDBIDPtr converts NullString to DBID pointer
func NullStringToDBIDPtr(s NullString) *DBID {
	if s == "" {
		return nil
	}
	dbid := DBID(s.String())
	return &dbid
}

// SQLNullStringToDBIDPtr converts sql.NullString to DBID pointer
func SQLNullStringToDBIDPtr(s sql.NullString) *DBID {
	if !s.Valid || s.String == "" {
		return nil
	}
	dbid := DBID(s.String)
	return &dbid
}

// DBIDSliceToStringSlice converts []DBID to []string
func DBIDSliceToStringSlice(ids []DBID) []string {
	result := make([]string, len(ids))
	for i, id := range ids {
		result[i] = id.String()
	}
	return result
}

// StringSliceToDBIDSlice converts []string to []DBID
func StringSliceToDBIDSlice(strs []string) []DBID {
	result := make([]DBID, len(strs))
	for i, str := range strs {
		result[i] = DBID(str)
	}
	return result
}

func StrPtrToNullStr(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{Valid: true, String: *s}
}

func NullStrToStr(s sql.NullString) string {
	if !s.Valid {
		return ""
	}
	return s.String
}

func NullStrToDBID(s sql.NullString) DBID {
	return DBID(NullStrToStr(s))
}

func DBIDToNullStr(id DBID) sql.NullString {
	if id == "" {
		return sql.NullString{}
	}
	return sql.NullString{Valid: true, String: id.String()}
}

func DBIDPtrToNullStr(id *DBID) sql.NullString {
	if id == nil || *id == "" {
		return sql.NullString{}
	}
	return sql.NullString{Valid: true, String: id.String()}
}

// NullInt64 represents an int64 that may be null in the DB.
type NullInt64 int64

// Int64 returns the int64 representation.
func (n NullInt64) Int64() int64 {
	return int64(n)
}

func (n NullInt64) String() string {
	return fmt.Sprint(n.Int64())
}

// Value implements the database/sql driver Valuer interface.
func (n NullInt64) Value() (driver.Value, error) {
	return n.Int64(), nil
}

// Scan implements the database/sql Scanner interface.
func (n *NullInt64) Scan(value interface{}) error {
	if value == nil {
		*n = NullInt64(0)
		return nil
	}
	*n = NullInt64(value.(int64))
	return nil
}

// NullInt32 represents an int32 that may be null in the DB.
type NullInt32 int32

// Int32 returns the int32 representation.
func (n NullInt32) Int32() int32 {
	return int32(n)
}

// Int returns the int representation.
func (n NullInt32) Int() int {
	return int(n)
}

func (n NullInt32) String() string {
	return fmt.Sprint(n.Int32())
}

// Value implements the database/sql driver Valuer interface.
func (n NullInt32) Value() (driver.Value, error) {
	return n.Int32(), nil
}

// Scan implements the database/sql Scanner interface.
func (n *NullInt32) Scan(value interface{}) error {
	if value == nil {
		*n = NullInt32(0)
		return nil
	}
	// database/sql spec says integer values should be returned as int64,
	// even if the underlying column is int32
	*n = NullInt32(value.(int64))
	return nil
}

// NullBool represents a bool that may be null in the DB
type NullBool bool

// Bool returns the bool representation
func (n NullBool) Bool() bool {
	return bool(n)
}

// BoolPointer returns a pointer to the bool value
func (n NullBool) BoolPointer() *bool {
	res := bool(n)
	return &res
}

func (n NullBool) String() string {
	return fmt.Sprint(n.Bool())
}

// Value implements the database/sql driver Valuer interface
func (n NullBool) Value() (driver.Value, error) {
	return n.Bool(), nil
}

// Scan implements the database/sql Scanner interface
func (n *NullBool) Scan(value interface{}) error {
	if value == nil {
		*n = NullBool(false)
		return nil
	}
	*n = NullBool(value.(bool))
	return nil
}

type CompleteIndex struct {
	Index  int `json:"start"`
	Length int `json:"end"`
}

func (c CompleteIndex) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func (c *CompleteIndex) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	return json.Unmarshal(value.([]uint8), c)
}

// GenerateID generates an application-wide unique ID using ksuid
func GenerateID() DBID {
	id, err := ksuid.NewRandom()
	if err != nil {
		panic(err)
	}
	return DBID(id.String())
}

// RemoveDuplicateDBIDs ensures that a slice of DBIDs has no repeated items
func RemoveDuplicateDBIDs(a []DBID) []DBID {
	result := make([]DBID, 0, len(a))
	m := map[DBID]bool{}

	for _, val := range a {
		if _, ok := m[val]; !ok {
			m[val] = true
			result = append(result, val)
		}
	}

	return result
}

// RemoveDuplicateAddresses ensures that a slice of addresses has no repeated items.
// Note: Requires EthereumAddress type to be defined in this package
func RemoveDuplicateAddresses(a []EthereumAddress) []EthereumAddress {
	result := make([]EthereumAddress, 0, len(a))
	m := map[EthereumAddress]bool{}

	for _, val := range a {
		if _, ok := m[val]; !ok {
			m[val] = true
			result = append(result, val)
		}
	}

	return result
}

// ContainsDBID checks if a DBID exists in a slice
func ContainsDBID(pSrc []DBID, pID DBID) bool {
	for _, v := range pSrc {
		if v == pID {
			return true
		}
	}
	return false
}

// ToDBIDs converts a slice of any type to a slice of DBIDs using a converter function
func ToDBIDs[T any](them []T, convert func(T) (DBID, error)) ([]DBID, error) {
	result := make([]DBID, len(them))
	for i, v := range them {
		d, err := convert(v)
		if err != nil {
			return nil, err
		}
		result[i] = d
	}
	return result, nil
}

// ToJSONB converts any value to pgtype.JSONB
func ToJSONB(v any) (pgtype.JSONB, error) {
	byt, err := json.Marshal(v)
	if err != nil {
		return pgtype.JSONB{}, err
	}
	ret := pgtype.JSONB{}
	err = ret.Set(byt)
	return ret, err
}

// JSONToJSONB converts persist.JSON to pgtype.JSONB
func JSONToJSONB(j JSON) pgtype.JSONB {
	if j == nil || len(j) == 0 {
		return pgtype.JSONB{Status: pgtype.Null}
	}
	return pgtype.JSONB{Bytes: []byte(j), Status: pgtype.Present}
}

// JSONBToJSON converts pgtype.JSONB to persist.JSON
func JSONBToJSON(j pgtype.JSONB) JSON {
	if j.Status != pgtype.Present || len(j.Bytes) == 0 {
		return nil
	}
	return JSON(j.Bytes)
}

type DarkMode int

const (
	DarkModeDisabled DarkMode = iota
	DarkModeEnabled
)

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (d *DarkMode) UnmarshalGQL(v interface{}) error {
	n, ok := v.(string)
	if !ok {
		return fmt.Errorf("darkMode must be a string")
	}

	switch strings.ToLower(n) {
	case "disabled":
		*d = DarkModeDisabled
	case "enabled":
		*d = DarkModeEnabled
	}
	return nil
}

// MarshalGQL implements the graphql.Marshaler interface
func (d DarkMode) MarshalGQL(w io.Writer) {
	switch d {
	case DarkModeDisabled:
		w.Write([]byte(`"Disabled"`))
	case DarkModeEnabled:
		w.Write([]byte(`"Enabled"`))
	}
}
