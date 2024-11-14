package goresponse

import (
	"fmt"
	"math"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/tlabdotcom/goencryption"
)

type (
	FilterOptions struct {
		Page       int      `param:"page" query:"page" form:"page" json:"page,omitempty" xml:"page,omitempty"`
		Limit      int      `param:"limit" query:"limit" form:"limit" json:"limit,omitempty" xml:"limit,omitempty"`
		Offset     *int     `param:"offset" query:"offset" form:"offset" json:"offset,omitempty" xml:"offset,omitempty"`
		Search     string   `param:"q" query:"q" form:"q" json:"q,omitempty" xml:"q,omitempty"`
		Dir        string   `param:"sort" query:"sort" form:"sort" json:"sort,omitempty" xml:"sort,omitempty"`
		SortBy     string   `param:"sort_by" query:"sort_by" form:"sort_by" json:"sort_by,omitempty" xml:"sort_by,omitempty"`
		StartDate  string   `param:"start_date" query:"start_date" form:"start_date" json:"start_date,omitempty" xml:"start_date,omitempty"`
		EndDate    string   `param:"end_date" query:"end_date" form:"end_date" json:"end_date,omitempty" xml:"end_date,omitempty"`
		Type       string   `param:"type" query:"type" form:"type" json:"type,omitempty" xml:"type,omitempty"`
		Status     string   `param:"status" query:"status" form:"status" json:"status,omitempty" xml:"status,omitempty"`
		Categories []string `param:"categories" query:"categories" form:"categories" json:"categories,omitempty" xml:"categories,omitempty"`
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

// handleFieldValue processes a single field value and returns its string representation
func handleFieldValue(fieldValue reflect.Value) (string, bool) {
	if fieldValue.Kind() == reflect.Ptr && fieldValue.IsNil() {
		return "", false
	}

	switch fieldValue.Kind() {
	case reflect.Ptr:
		value := fmt.Sprintf("%v", fieldValue.Elem().Interface())
		return value, value != ""

	case reflect.String:
		value := fieldValue.String()
		return value, value != ""

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value := fmt.Sprintf("%d", fieldValue.Int())
		return value, value != "0"

	case reflect.Slice:
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

	return "", false
}

// formatCacheKeyField formats a field name and value into a cache key segment
func formatCacheKeyField(fieldName, fieldValue string) string {
	return fmt.Sprintf("%s:%s", fieldName, fieldValue)
}

// GenerateCacheKey generates a unique cache key based on filter values
func (filter FilterOptions) GenerateCacheKey(redisKeyPrefix string) string {
	sortedFields := make([]string, 0)
	v := reflect.ValueOf(filter)
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i).Tag.Get("query")
		if field == "" {
			continue
		}

		fieldValue := v.Field(i)
		value, hasValue := handleFieldValue(fieldValue)
		if hasValue {
			sortedFields = append(sortedFields, formatCacheKeyField(field, value))
		}
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
