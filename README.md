# Pkg for golang default response

Maintainer: [jiharal](https://github.com/jiharal)

![Go CI](https://github.com/tlabdotcom/goresponse/actions/workflows/go.yml/badge.svg)

[![Go Report Card](https://goreportcard.com/badge/github.com/tlabdotcom/goresponse)](https://goreportcard.com/report/github.com/tlabdotcom/goresponse)

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

## notes

before push please check your code using

gocyclo

```shell
gocyclo -over 10 .
```

golangci-lint

```shell
golangci-lint run
```

go test

```shell
go test ./... -coverprofile=coverage.out
```
