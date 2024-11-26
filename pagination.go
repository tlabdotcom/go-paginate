package goresponse

import (
	"fmt"
	"math"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/tlabdotcom/goencryption"
)

type (
	FilterOptions struct {
		Page          int                    `param:"page" query:"page" form:"page" json:"page,omitempty" xml:"page,omitempty"`
		Limit         int                    `param:"limit" query:"limit" form:"limit" json:"limit,omitempty" xml:"limit,omitempty"`
		Offset        *int                   `param:"offset" query:"offset" form:"offset" json:"offset,omitempty" xml:"offset,omitempty"`
		Search        string                 `param:"q" query:"q" form:"q" json:"q,omitempty" xml:"q,omitempty"`
		Dir           string                 `param:"sort" query:"sort" form:"sort" json:"sort,omitempty" xml:"sort,omitempty"`
		SortBy        string                 `param:"sort_by" query:"sort_by" form:"sort_by" json:"sort_by,omitempty" xml:"sort_by,omitempty"`
		StartDate     string                 `param:"start_date" query:"start_date" form:"start_date" json:"start_date,omitempty" xml:"start_date,omitempty"`
		EndDate       string                 `param:"end_date" query:"end_date" form:"end_date" json:"end_date,omitempty" xml:"end_date,omitempty"`
		Type          string                 `param:"type" query:"type" form:"type" json:"type,omitempty" xml:"type,omitempty"`
		Status        string                 `param:"status" query:"status" form:"status" json:"status,omitempty" xml:"status,omitempty"`
		Categories    []string               `param:"categories" query:"categories" form:"categories" json:"categories,omitempty" xml:"categories,omitempty"`
		DynamicFields map[string]interface{} `json:"-"`
	}
	PaginatedResponse struct {
		TotalData   int            `json:"total_data,omitempty"`
		TotalPage   int            `json:"total_page,omitempty"`
		CurrentPage int            `json:"current_page,omitempty"`
		PageSize    int            `json:"page_size,omitempty"`
		Data        interface{}    `json:"data,omitempty"`
		Filters     *FilterOptions `json:"filters,omitempty"`
	}
)

func (f *FilterOptions) GetDynamicField(key string) (interface{}, bool) {
	if f.DynamicFields == nil {
		return nil, false
	}
	value, exists := f.DynamicFields[key]
	return value, exists
}

func (f *FilterOptions) SetDynamicField(key string, value interface{}) {
	if f.DynamicFields == nil {
		f.DynamicFields = make(map[string]interface{})
	}
	f.DynamicFields[key] = value
}

// Add this helper function to validate and parse UUID
func parseUUID(value string) (uuid.UUID, error) {
	if value == "" {
		return uuid.Nil, nil
	}
	return uuid.Parse(value)
}

func handleKnownFields(filter *FilterOptions, values url.Values, knownParams map[string]struct{}) error {
	t := reflect.TypeOf(*filter)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Name == "DynamicFields" {
			continue
		}

		queryTag := field.Tag.Get("query")
		if queryTag == "" {
			continue
		}

		knownParams[queryTag] = struct{}{}

		if value := values.Get(queryTag); value != "" {
			v := reflect.ValueOf(filter).Elem()
			fieldValue := v.Field(i)

			if err := setFieldFromString(fieldValue, value); err != nil {
				return fmt.Errorf("error setting field %s: %v", field.Name, err)
			}
		}
	}
	return nil
}

func ParseURLValues(values url.Values) (*FilterOptions, error) {
	filter := &FilterOptions{
		DynamicFields: make(map[string]interface{}),
	}

	knownParams := make(map[string]struct{})

	if err := handleKnownFields(filter, values, knownParams); err != nil {
		return nil, err
	}

	// Handle dynamic fields
	for param, paramValues := range values {
		if _, isKnown := knownParams[param]; !isKnown && len(paramValues) > 0 {
			filter.DynamicFields[param] = handleDynamicField(paramValues)
		}
	}

	return filter.Validate(), nil
}

func handleDynamicUUIDValue(value string) (interface{}, bool) {
	if parsedUUID, err := parseUUID(value); err == nil {
		return parsedUUID, true
	}
	return nil, false
}

func handleDynamicUUIDSlice(values []string) (interface{}, bool) {
	uuids := make([]uuid.UUID, 0, len(values))
	for _, v := range values {
		if parsedUUID, err := parseUUID(v); err == nil {
			uuids = append(uuids, parsedUUID)
		} else {
			return nil, false
		}
	}
	return uuids, true
}

func handleDynamicField(values []string) interface{} {
	if len(values) == 1 {
		if uuidValue, ok := handleDynamicUUIDValue(values[0]); ok {
			return uuidValue
		}
		return values[0]
	}

	if uuidSlice, ok := handleDynamicUUIDSlice(values); ok {
		return uuidSlice
	}
	return values
}

func handleUUIDField(field reflect.Value, value string) error {
	if parsedUUID, err := parseUUID(value); err == nil {
		field.Set(reflect.ValueOf(parsedUUID))
		return nil
	} else {
		return fmt.Errorf("invalid UUID format: %v", err)
	}
}

func handleNumericField(field reflect.Value, value string) error {
	if value == "" {
		return nil
	}
	intValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return err
	}
	field.SetInt(intValue)
	return nil
}

func handleStringSlice(field reflect.Value, value string) error {
	values := strings.Split(value, ",")
	slice := reflect.MakeSlice(field.Type(), len(values), len(values))
	for i, v := range values {
		slice.Index(i).SetString(strings.TrimSpace(v))
	}
	field.Set(slice)
	return nil
}

func handleUUIDSlice(field reflect.Value, value string) error {
	values := strings.Split(value, ",")
	slice := reflect.MakeSlice(field.Type(), 0, len(values))
	for _, v := range values {
		if parsedUUID, err := parseUUID(strings.TrimSpace(v)); err == nil {
			slice = reflect.Append(slice, reflect.ValueOf(parsedUUID))
		} else {
			return fmt.Errorf("invalid UUID in slice: %v", err)
		}
	}
	field.Set(slice)
	return nil
}

func setFieldFromString(field reflect.Value, value string) error {
	// Handle UUID type
	if field.Type().String() == "uuid.UUID" {
		return handleUUIDField(field, value)
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(value)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return handleNumericField(field, value)

	case reflect.Ptr:
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return setFieldFromString(field.Elem(), value)

	case reflect.Slice:
		if field.Type().Elem().Kind() == reflect.String {
			return handleStringSlice(field, value)
		} else if field.Type().Elem() == reflect.TypeOf(uuid.UUID{}) {
			return handleUUIDSlice(field, value)
		}
	}
	return nil
}

// handleSliceValue handles slice type values
func handleSliceValue(fieldValue reflect.Value) (string, bool) {
	if fieldValue.IsNil() || fieldValue.Len() == 0 {
		return "", false
	}
	values := make([]string, fieldValue.Len())
	for i := 0; i < fieldValue.Len(); i++ {
		values[i] = fmt.Sprintf("%v", fieldValue.Index(i).Interface())
	}
	sort.Strings(values)
	return strings.Join(values, ","), true
}

// handlePtrValue handles pointer type values
func handlePtrValue(fieldValue reflect.Value) (string, bool) {
	value := fmt.Sprintf("%v", fieldValue.Elem().Interface())
	return value, value != ""
}

// handleNumericValue handles numeric type values
func handleNumericValue(fieldValue reflect.Value) (string, bool) {
	value := fmt.Sprintf("%d", fieldValue.Int())
	return value, value != "0"
}

// handleUUIDValue handles UUID type values
func handleUUIDValue(fieldValue reflect.Value) (string, bool) {
	value := fieldValue.Interface().(uuid.UUID)
	if value == uuid.Nil {
		return "", false
	}
	return value.String(), true
}

// handleMapValue handles map type values
func handleMapValue(fieldValue reflect.Value) (string, bool) {
	if fieldValue.IsNil() || fieldValue.Len() == 0 {
		return "", false
	}
	return "", true
}

// handleFieldValue processes a single field value and returns its string representation
func handleFieldValue(fieldValue reflect.Value) (string, bool) {
	if fieldValue.Kind() == reflect.Ptr && fieldValue.IsNil() {
		return "", false
	}

	// Handle UUID type
	if fieldValue.Type().String() == "uuid.UUID" {
		return handleUUIDValue(fieldValue)
	}

	// Handle other types based on Kind
	switch fieldValue.Kind() {
	case reflect.Ptr:
		return handlePtrValue(fieldValue)

	case reflect.String:
		value := fieldValue.String()
		return value, value != ""

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return handleNumericValue(fieldValue)

	case reflect.Slice:
		return handleSliceValue(fieldValue)

	case reflect.Map:
		return handleMapValue(fieldValue)
	}

	return "", false
}

// formatCacheKeyField formats a field name and value into a cache key segment
func formatCacheKeyField(fieldName, fieldValue string) string {
	return fmt.Sprintf("%s:%s", fieldName, fieldValue)
}

func sortDynamicFields(fields map[string]interface{}) []string {
	// Get all dynamic field keys
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Create sorted key-value pairs
	sorted := make([]string, 0, len(fields))
	for _, k := range keys {
		v := fields[k]
		var value string

		switch v := v.(type) {
		case uuid.UUID:
			value = v.String()
		case []uuid.UUID:
			uuidStrs := make([]string, len(v))
			for i, u := range v {
				uuidStrs[i] = u.String()
			}
			sort.Strings(uuidStrs)
			value = strings.Join(uuidStrs, ",")
		case []string:
			sort.Strings(v)
			value = strings.Join(v, ",")
		case []interface{}:
			strVals := make([]string, len(v))
			for i, val := range v {
				strVals[i] = fmt.Sprintf("%v", val)
			}
			sort.Strings(strVals)
			value = strings.Join(strVals, ",")
		default:
			value = fmt.Sprintf("%v", v)
		}

		if value != "" {
			sorted = append(sorted, formatCacheKeyField(k, value))
		}
	}

	return sorted
}

// GenerateCacheKey generates a unique cache key based on filter values
func (filter FilterOptions) GenerateCacheKey(redisKeyPrefix string) string {
	sortedFields := make([]string, 0)
	v := reflect.ValueOf(filter)
	t := v.Type()

	// Handle standard fields
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		if field.Name == "DynamicFields" {
			continue
		}

		queryTag := field.Tag.Get("query")
		if queryTag == "" {
			continue
		}

		fieldValue := v.Field(i)
		value, hasValue := handleFieldValue(fieldValue)
		if hasValue {
			sortedFields = append(sortedFields, formatCacheKeyField(queryTag, value))
		}
	}

	// Handle dynamic fields
	if len(filter.DynamicFields) > 0 {
		dynamicFields := sortDynamicFields(filter.DynamicFields)
		sortedFields = append(sortedFields, dynamicFields...)
	}

	sort.Strings(sortedFields)
	concatenated := strings.Join(sortedFields, ":")
	hashValue := goencryption.Sha256Hash([]byte(concatenated))

	return redisKeyPrefix + "list:" + hashValue
}

// GetMaxLimitFromEnv reads the maximum limit from environment variables, with a default of 100
func GetMaxLimitFromEnv() int {
	// Get the environment variable MAX_LIMIT, if not set default to 100
	envLimit := os.Getenv("MAX_LIMIT_PAGINATE")
	maxLimit, err := strconv.Atoi(envLimit)
	if err != nil || maxLimit < 1 {
		maxLimit = 100 // Default to 100 if the environment variable is not set or invalid
	}
	return maxLimit
}

func (filter *FilterOptions) Validate() *FilterOptions {
	// Ensure Page is always at least 1
	if filter.Page < 1 {
		filter.Page = 1
	}

	// Get the max limit from environment variable or use the default of 100
	maxLimit := GetMaxLimitFromEnv()

	// Ensure Limit is at least 1 and at most maxLimit
	if filter.Limit < 1 {
		filter.Limit = 1
	} else if filter.Limit > maxLimit {
		filter.Limit = maxLimit
	}

	// Clean up Search query (trim whitespace)
	filter.Search = strings.TrimSpace(filter.Search)

	// Normalize Sort Direction (Dir)
	switch strings.ToLower(filter.Dir) {
	case "asc":
		filter.Dir = "ASC"
	case "desc":
		filter.Dir = "DESC"
	default:
		filter.Dir = "DESC" // Default to DESC
	}

	// Calculate Offset if not already set
	if filter.Offset == nil {
		offset := (filter.Page - 1) * filter.Limit
		filter.Offset = &offset
	}
	// Clean up Type and Status
	filter.Type = strings.TrimSpace(filter.Type)
	filter.Status = strings.TrimSpace(filter.Status)

	// Clean up Dates
	filter.StartDate = strings.TrimSpace(filter.StartDate)
	filter.EndDate = strings.TrimSpace(filter.EndDate)

	// Clean up Categories
	if len(filter.Categories) > 0 {
		cleanCategories := make([]string, 0)
		for _, category := range filter.Categories {
			if trimmed := strings.TrimSpace(category); trimmed != "" {
				cleanCategories = append(cleanCategories, trimmed)
			}
		}
		filter.Categories = cleanCategories
	}

	return filter
}

func GeneratePaginatedResponse(data interface{}, totalData int, filter *FilterOptions) *PaginatedResponse {
	totalPage := int(math.Ceil(float64(totalData) / float64(filter.Limit)))

	return &PaginatedResponse{
		TotalData:   totalData,
		TotalPage:   totalPage,
		CurrentPage: filter.Page,
		PageSize:    filter.Limit,
		Data:        data,
		Filters:     filter,
	}
}

// Helper method to get UUID from dynamic fields
func (f *FilterOptions) GetDynamicUUID(key string) (uuid.UUID, bool) {
	if value, exists := f.DynamicFields[key]; exists {
		if uuid, ok := value.(uuid.UUID); ok {
			return uuid, true
		}
	}
	return uuid.Nil, false
}

// Helper method to get UUID slice from dynamic fields
func (f *FilterOptions) GetDynamicUUIDs(key string) ([]uuid.UUID, bool) {
	if value, exists := f.DynamicFields[key]; exists {
		if uuids, ok := value.([]uuid.UUID); ok {
			return uuids, true
		}
	}
	return nil, false
}

// For Echo framework
func HandleFilterOptionsEcho(c echo.Context) (*FilterOptions, error) {
	return ParseURLValues(c.QueryParams())
}

// For Gin framework #TODO
// func HandleFilterOptionsGin(c *gin.Context) (*FilterOptions, error) {
// 	return ParseURLValues(c.Request.URL.Query())
// }

// For Fiber framework #TODO
// func HandleFilterOptionsFiber(c *fiber.Ctx) (*FilterOptions, error) {
// 	values, err := url.ParseQuery(string(c.Request().URI().QueryString()))
// 	if err != nil {
// 		return nil, err
// 	}
// 	return ParseURLValues(values)
// }
