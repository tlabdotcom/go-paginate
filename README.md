# Pkg for golang default response



## Env for paginations
```shell
MAX_LIMIT_PAGINATE=150
```
by default is 100



Example Usage:
For Validation Error Response:
```go
validationErr := validator.New().Struct(data)
if validationErr != nil {
    response := NewStandardErrorResponse(http.StatusUnprocessableEntity)
    response.AddError(validationErr).JSON(c)
}
```
For Database Error Response:
```go
dbErr := db.QueryRowContext(ctx, query).Scan(&result)
if dbErr != nil {
    response := NewStandardErrorResponse(http.StatusInternalServerError)
    response.AddError(dbErr).JSON(c)
}
```
For a Successful Response:
```go
response := GenerateSingleDataResponse(data, "Data retrieved successfully", http.StatusOK)

```
For a paginations
```go
// Query the user data from the database or service
	users, totalRecords, err := queryUsers(filter.Search, filter.Limit, *filter.Offset)
	if err != nil {
		response := NewStandardErrorResponse(http.StatusUnprocessableEntity)
    response.AddError(validationErr).JSON(c)
	}
	// Generate a paginated response
	paginatedResponse := GeneratePaginatedResponse(users, totalRecords, filter)

	return c.JSON(http.StatusOK, paginatedResponse)
```