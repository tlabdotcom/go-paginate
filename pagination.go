package response

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
		Page   int    `param:"page" query:"page" form:"page" json:"page,omitempty" xml:"page,omitempty"`
		Limit  int    `param:"limit" query:"limit" form:"limit" json:"limit,omitempty" xml:"limit,omitempty"`
		Offset *int   `param:"offset" query:"offset" form:"offset" json:"offset,omitempty" xml:"offset,omitempty"`
		Search string `param:"q" query:"q" form:"q" json:"q,omitempty" xml:"q,omitempty"`
		Dir    string `param:"sort" query:"sort" form:"sort" json:"sort,omitempty" xml:"sort,omitempty"`
		SortBy string `param:"sort_by" query:"sort_by" form:"sort_by" json:"sort_by,omitempty" xml:"sort_by,omitempty"`
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

func (filter FilterOptions) GenerateCacheKey(redisKeyPrefix string) string {
	sortedFields := make([]string, 0)
	v := reflect.ValueOf(filter)

	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i).Tag.Get("query")
		fieldValue := v.Field(i)

		// Skip if no field name (empty tag)
		if field == "" {
			continue
		}

		// Handle based on the field type
		switch fieldValue.Kind() {
		case reflect.Ptr: // Handle pointer fields like *int
			if !fieldValue.IsNil() {
				value := fmt.Sprintf("%v", fieldValue.Elem().Interface())
				if value != "" {
					sortedFields = append(sortedFields, fmt.Sprintf("%s:%s", field, value))
				}
			}
		case reflect.String: // Handle string fields
			value := fieldValue.String()
			if value != "" {
				sortedFields = append(sortedFields, fmt.Sprintf("%s:%s", field, value))
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64: // Handle integer fields
			value := fmt.Sprintf("%d", fieldValue.Int())
			if value != "0" { // Skip zero values for integers
				sortedFields = append(sortedFields, fmt.Sprintf("%s:%s", field, value))
			}
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

	return filter
}

func GeneratePaginatedResponse(data interface{}, totalData int, filter *FilterOptions) *PaginatedResponse {
	// Calculate total pages, rounding up
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
