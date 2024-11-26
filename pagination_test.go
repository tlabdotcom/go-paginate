package goresponse

import (
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestParseURLValues(t *testing.T) {
	tests := []struct {
		name        string
		urlValues   url.Values
		want        *FilterOptions
		wantErr     bool
		errContains string
	}{
		{
			name: "basic fields",
			urlValues: url.Values{
				"page":    []string{"1"},
				"limit":   []string{"10"},
				"q":       []string{"search text"},
				"sort":    []string{"desc"},
				"sort_by": []string{"created_at"},
			},
			want: &FilterOptions{
				Page:          1,
				Limit:         10,
				Search:        "search text",
				Dir:           "DESC",
				SortBy:        "created_at",
				Offset:        intPtr(0),
				DynamicFields: make(map[string]interface{}),
			},
			wantErr: false,
		},
		{
			name: "with categories",
			urlValues: url.Values{
				"page":       []string{"1"},
				"categories": []string{"cat1,cat2,cat3"},
			},
			want: &FilterOptions{
				Page:          1,
				Limit:         1,
				Categories:    []string{"cat1", "cat2", "cat3"},
				Dir:           "DESC",
				Offset:        intPtr(0),
				DynamicFields: make(map[string]interface{}),
			},
			wantErr: false,
		},
		{
			name: "with dynamic UUID field",
			urlValues: url.Values{
				"page":    []string{"1"},
				"user_id": []string{"123e4567-e89b-12d3-a456-426614174000"},
			},
			want: &FilterOptions{
				Page:   1,
				Limit:  1,
				Dir:    "DESC",
				Offset: intPtr(0),
				DynamicFields: map[string]interface{}{
					"user_id": uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
				},
			},
			wantErr: false,
		},
		{
			name: "with invalid page number",
			urlValues: url.Values{
				"page":  []string{"invalid"},
				"limit": []string{"10"},
			},
			wantErr:     true,
			errContains: "error setting field Page",
		},
		{
			name: "with multiple UUID values",
			urlValues: url.Values{
				"user_ids": []string{
					"123e4567-e89b-12d3-a456-426614174000",
					"987fcdeb-51a2-43d7-9012-345678901234",
				},
			},
			want: &FilterOptions{
				Page:   1,
				Limit:  1,
				Dir:    "DESC",
				Offset: intPtr(0),
				DynamicFields: map[string]interface{}{
					"user_ids": []uuid.UUID{
						uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
						uuid.MustParse("987fcdeb-51a2-43d7-9012-345678901234"),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "with dates",
			urlValues: url.Values{
				"start_date": []string{"2024-01-01"},
				"end_date":   []string{"2024-12-31"},
			},
			want: &FilterOptions{
				Page:          1,
				Limit:         1,
				Dir:           "DESC",
				Offset:        intPtr(0),
				StartDate:     "2024-01-01",
				EndDate:       "2024-12-31",
				DynamicFields: make(map[string]interface{}),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseURLValues(tt.urlValues)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			assert.NoError(t, err)

			// Use a custom comparison that ignores pointer addresses
			assertEqual(t, tt.want, got)
		})
	}
}

// Custom assertion function that compares FilterOptions while ignoring pointer addresses
func assertEqual(t *testing.T, expected, actual *FilterOptions) {
	t.Helper()

	// Compare non-pointer fields
	assert.Equal(t, expected.Page, actual.Page)
	assert.Equal(t, expected.Limit, actual.Limit)
	assert.Equal(t, expected.Search, actual.Search)
	assert.Equal(t, expected.Dir, actual.Dir)
	assert.Equal(t, expected.SortBy, actual.SortBy)
	assert.Equal(t, expected.StartDate, actual.StartDate)
	assert.Equal(t, expected.EndDate, actual.EndDate)
	assert.Equal(t, expected.Type, actual.Type)
	assert.Equal(t, expected.Status, actual.Status)
	assert.Equal(t, expected.Categories, actual.Categories)

	// Compare Offset values, not pointers
	if expected.Offset == nil {
		assert.Nil(t, actual.Offset)
	} else {
		assert.NotNil(t, actual.Offset)
		assert.Equal(t, *expected.Offset, *actual.Offset)
	}

	// Compare DynamicFields
	assert.Equal(t, expected.DynamicFields, actual.DynamicFields)
}

// Rest of the test file remains the same...

func TestFilterOptions_Validate(t *testing.T) {
	tests := []struct {
		name   string
		filter *FilterOptions
		want   *FilterOptions
	}{
		{
			name: "normalize page and limit",
			filter: &FilterOptions{
				Page:          0,
				Limit:         0,
				DynamicFields: make(map[string]interface{}),
			},
			want: &FilterOptions{
				Page:          1,
				Limit:         1,
				Dir:           "DESC",
				Offset:        intPtr(0),
				DynamicFields: make(map[string]interface{}),
			},
		},
		{
			name: "normalize sort direction",
			filter: &FilterOptions{
				Page:          1,
				Limit:         10,
				Dir:           "asc",
				DynamicFields: make(map[string]interface{}),
			},
			want: &FilterOptions{
				Page:          1,
				Limit:         10,
				Dir:           "ASC",
				Offset:        intPtr(0),
				DynamicFields: make(map[string]interface{}),
			},
		},
		{
			name: "trim whitespace",
			filter: &FilterOptions{
				Page:          1,
				Limit:         10,
				Search:        "  test search  ",
				Type:          "  type1  ",
				Status:        "  active  ",
				StartDate:     "  2024-01-01  ",
				EndDate:       "  2024-12-31  ",
				Categories:    []string{"  cat1  ", "  cat2  "},
				DynamicFields: make(map[string]interface{}),
			},
			want: &FilterOptions{
				Page:          1,
				Limit:         10,
				Search:        "test search",
				Type:          "type1",
				Status:        "active",
				StartDate:     "2024-01-01",
				EndDate:       "2024-12-31",
				Categories:    []string{"cat1", "cat2"},
				Dir:           "DESC",
				Offset:        intPtr(0),
				DynamicFields: make(map[string]interface{}),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.Validate()
			assertEqual(t, tt.want, got)
		})
	}
}

func TestFilterOptions_GetDynamicUUIDs(t *testing.T) {
	uuid1 := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	uuid2 := uuid.MustParse("987fcdeb-51a2-43d7-9012-345678901234")

	tests := []struct {
		name   string
		fields map[string]interface{}
		key    string
		want   []uuid.UUID
		wantOk bool
	}{
		{
			name: "valid UUID slice",
			fields: map[string]interface{}{
				"user_ids": []uuid.UUID{uuid1, uuid2},
			},
			key:    "user_ids",
			want:   []uuid.UUID{uuid1, uuid2},
			wantOk: true,
		},
		{
			name: "non-existent key",
			fields: map[string]interface{}{
				"other_ids": []uuid.UUID{uuid1, uuid2},
			},
			key:    "user_ids",
			want:   nil,
			wantOk: false,
		},
		{
			name: "wrong type",
			fields: map[string]interface{}{
				"user_ids": []string{"not-a-uuid"},
			},
			key:    "user_ids",
			want:   nil,
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FilterOptions{
				DynamicFields: tt.fields,
			}
			got, ok := f.GetDynamicUUIDs(tt.key)
			assert.Equal(t, tt.wantOk, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}
func TestGeneratePaginatedResponse(t *testing.T) {
	tests := []struct {
		name      string
		data      interface{}
		totalData int
		filter    *FilterOptions
		want      *PaginatedResponse
	}{
		{
			name:      "basic pagination",
			data:      []string{"item1", "item2"},
			totalData: 10,
			filter: &FilterOptions{
				Page:  1,
				Limit: 2,
			},
			want: &PaginatedResponse{
				TotalData:   10,
				TotalPage:   5,
				CurrentPage: 1,
				PageSize:    2,
				Data:        []string{"item1", "item2"},
				Filters: &FilterOptions{
					Page:  1,
					Limit: 2,
				},
			},
		},
		{
			name:      "uneven division",
			data:      []string{"item1"},
			totalData: 3,
			filter: &FilterOptions{
				Page:  2,
				Limit: 2,
			},
			want: &PaginatedResponse{
				TotalData:   3,
				TotalPage:   2,
				CurrentPage: 2,
				PageSize:    2,
				Data:        []string{"item1"},
				Filters: &FilterOptions{
					Page:  2,
					Limit: 2,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GeneratePaginatedResponse(tt.data, tt.totalData, tt.filter)
			assert.Equal(t, tt.want, got)
		})
	}
}
func TestGenerateCacheKey(t *testing.T) {
	tests := []struct {
		name                  string
		filter                FilterOptions
		redisKeyPrefix        string
		shouldBeDifferentFrom []FilterOptions
	}{
		{
			name: "basic cache key",
			filter: FilterOptions{
				Page:          1,
				Limit:         10,
				Search:        "test",
				DynamicFields: map[string]interface{}{},
			},
			redisKeyPrefix: "test:",
			shouldBeDifferentFrom: []FilterOptions{
				{
					Page:          2,
					Limit:         10,
					Search:        "test",
					DynamicFields: map[string]interface{}{},
				},
				{
					Page:          1,
					Limit:         10,
					Search:        "different",
					DynamicFields: map[string]interface{}{},
				},
			},
		},
		{
			name: "with dynamic UUID field",
			filter: FilterOptions{
				Page:  1,
				Limit: 10,
				DynamicFields: map[string]interface{}{
					"user_id": uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
				},
			},
			redisKeyPrefix: "test:",
			shouldBeDifferentFrom: []FilterOptions{
				{
					Page:  1,
					Limit: 10,
					DynamicFields: map[string]interface{}{
						"user_id": uuid.MustParse("987fcdeb-51a2-43d7-9012-345678901234"),
					},
				},
			},
		},
		{
			name: "with UUID array",
			filter: FilterOptions{
				Page:  1,
				Limit: 10,
				DynamicFields: map[string]interface{}{
					"user_ids": []uuid.UUID{
						uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
						uuid.MustParse("987fcdeb-51a2-43d7-9012-345678901234"),
					},
				},
			},
			redisKeyPrefix: "test:",
			shouldBeDifferentFrom: []FilterOptions{
				{
					Page:  1,
					Limit: 10,
					DynamicFields: map[string]interface{}{
						"user_ids": []uuid.UUID{
							uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
						},
					},
				},
			},
		},
		{
			name: "with multiple dynamic fields",
			filter: FilterOptions{
				Page:  1,
				Limit: 10,
				DynamicFields: map[string]interface{}{
					"status":   "active",
					"user_id":  uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
					"tags":     []string{"tag1", "tag2"},
					"enabled":  true,
					"priority": 1,
				},
			},
			redisKeyPrefix: "test:",
			shouldBeDifferentFrom: []FilterOptions{
				{
					Page:  1,
					Limit: 10,
					DynamicFields: map[string]interface{}{
						"status":   "inactive",
						"user_id":  uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
						"tags":     []string{"tag1", "tag2"},
						"enabled":  true,
						"priority": 1,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test consistency
			key1 := tt.filter.GenerateCacheKey(tt.redisKeyPrefix)
			key2 := tt.filter.GenerateCacheKey(tt.redisKeyPrefix)
			assert.Equal(t, key1, key2, "Same filter should generate same key")

			// Test prefix
			assert.True(t, strings.HasPrefix(key1, tt.redisKeyPrefix+"list:"),
				"Cache key should have correct prefix")

			// Test different filters generate different keys
			for _, differentFilter := range tt.shouldBeDifferentFrom {
				differentKey := differentFilter.GenerateCacheKey(tt.redisKeyPrefix)
				assert.NotEqual(t, key1, differentKey,
					"Different filters should generate different keys")
			}
		})
	}
}

// Helper function to test cache key ordering consistency
func TestGenerateCacheKeyOrdering(t *testing.T) {
	filter1 := FilterOptions{
		DynamicFields: map[string]interface{}{
			"a": "1",
			"b": "2",
		},
	}

	filter2 := FilterOptions{
		DynamicFields: map[string]interface{}{
			"b": "2",
			"a": "1",
		},
	}

	key1 := filter1.GenerateCacheKey("test:")
	key2 := filter2.GenerateCacheKey("test:")

	assert.Equal(t, key1, key2, "Different field ordering should generate same key")
}

// Test cache key with nil dynamic fields
func TestGenerateCacheKeyNilDynamicFields(t *testing.T) {
	filter1 := FilterOptions{
		Page:   1,
		Limit:  10,
		Search: "test",
	}

	filter2 := FilterOptions{
		Page:          1,
		Limit:         10,
		Search:        "test",
		DynamicFields: make(map[string]interface{}),
	}

	key1 := filter1.GenerateCacheKey("test:")
	key2 := filter2.GenerateCacheKey("test:")

	assert.Equal(t, key1, key2, "Nil and empty dynamic fields should generate same key")
}

func TestHandleSliceValue(t *testing.T) {
	tests := []struct {
		name      string
		value     interface{}
		want      string
		wantValid bool
	}{
		{
			name:      "string slice",
			value:     []string{"c", "b", "a"},
			want:      "a,b,c",
			wantValid: true,
		},
		{
			name:      "empty slice",
			value:     []string{},
			want:      "",
			wantValid: false,
		},
		{
			name:      "nil slice",
			value:     []string(nil),
			want:      "",
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := reflect.ValueOf(tt.value)
			got, valid := handleSliceValue(val)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantValid, valid)
		})
	}
}

func TestHandleNumericValue(t *testing.T) {
	tests := []struct {
		name      string
		value     interface{}
		want      string
		wantValid bool
	}{
		{
			name:      "positive int",
			value:     42,
			want:      "42",
			wantValid: true,
		},
		{
			name:      "zero",
			value:     0,
			want:      "0",
			wantValid: false,
		},
		{
			name:      "negative int",
			value:     -1,
			want:      "-1",
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := reflect.ValueOf(tt.value)
			got, valid := handleNumericValue(val)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantValid, valid)
		})
	}
}

func TestHandleUUIDValue(t *testing.T) {
	validUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	tests := []struct {
		name      string
		value     uuid.UUID
		want      string
		wantValid bool
	}{
		{
			name:      "valid uuid",
			value:     validUUID,
			want:      validUUID.String(),
			wantValid: true,
		},
		{
			name:      "nil uuid",
			value:     uuid.Nil,
			want:      "",
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := reflect.ValueOf(tt.value)
			got, valid := handleUUIDValue(val)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantValid, valid)
		})
	}
}

func TestHandleMapValue(t *testing.T) {
	tests := []struct {
		name      string
		value     interface{}
		want      string
		wantValid bool
	}{
		{
			name: "non-empty map",
			value: map[string]string{
				"key": "value",
			},
			want:      "",
			wantValid: true,
		},
		{
			name:      "empty map",
			value:     map[string]string{},
			want:      "",
			wantValid: false,
		},
		{
			name:      "nil map",
			value:     map[string]string(nil),
			want:      "",
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := reflect.ValueOf(tt.value)
			got, valid := handleMapValue(val)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantValid, valid)
		})
	}
}

func TestHandleFieldValue(t *testing.T) {
	validUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	tests := []struct {
		name      string
		value     interface{}
		want      string
		wantValid bool
	}{
		{
			name:      "string value",
			value:     "test",
			want:      "test",
			wantValid: true,
		},
		{
			name:      "empty string",
			value:     "",
			want:      "",
			wantValid: false,
		},
		{
			name:      "integer value",
			value:     42,
			want:      "42",
			wantValid: true,
		},
		{
			name:      "uuid value",
			value:     validUUID,
			want:      validUUID.String(),
			wantValid: true,
		},
		{
			name:      "string slice",
			value:     []string{"a", "b", "c"},
			want:      "a,b,c",
			wantValid: true,
		},
		{
			name: "map value",
			value: map[string]string{
				"key": "value",
			},
			want:      "",
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := reflect.ValueOf(tt.value)
			got, valid := handleFieldValue(val)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantValid, valid)
		})
	}
}

// Helper function for tests
func intPtr(i int) *int {
	return &i
}
