# Primer

```shell
# Quick test with basic output
go run main.go -quick

# Full test with verbose output in JSON format
go run main.go -config config.json -format json -verbose -output results.json

# Compare with existing constants
go run main.go -quick -compare existing_constants.json

# Generate CSV output
go run main.go -format csv -output results.csv

# Run all tests
go test ./constants/...
```

Then just cross your fingers.
