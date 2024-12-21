## Setup

Run api
```
cd backend
go run cmd/*.go api           
```

Run migration
```
go run cmd/*.go migration --action up   // action up to update database
go run cmd/*.go migration --action down // action down to revert database
```

Generate bob
```
go run github.com/stephenafamo/bob/gen/bobgen-psql@v0.21.1 -c ./internal/config/bobgen.yaml
```

Clean and lint code
```
gofumpt -l -w .
golangci-lint run ./...
```