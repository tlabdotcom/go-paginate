# Dynamic Params

Key features of this implementation:

1. Automatically handles both standard and dynamic fields from URL parameters
2. Supports multiple values for the same parameter
3. Type conversion for different field types (int,uuid, string, slices, etc.)
4. Framework-agnostic core implementation with specific adapters for popular frameworks
5. Maintains validation through the existing Validate() method
   Preserves all existing FilterOptions functionality

```go
// Example using Echo framework
func YourHandler(c echo.Context) error {
    // Get filter with both standard and dynamic fields from URL
    filter, err := HandleFilterOptionsEcho(c)
    if err != nil {
        return c.JSON(http.StatusBadRequest, err.Error())
    }

    // Access a standard field
    page := filter.Page

    // Access a dynamic field (like audio_id from URL)
    if audioID, exists := filter.GetDynamicField("audio_id"); exists {
        // Use audioID
        fmt.Printf("Audio ID: %v\n", audioID)
    }

    // Use the filter for your database query or other operations...

    return c.JSON(http.StatusOK, result)
}
``
```
