module github.com/xyn-pos/services/payment

go 1.26

replace (
	github.com/xyn-pos/gen => ../../gen
	github.com/xyn-pos/shared => ../../shared/go
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.10.0 // indirect
	github.com/mfridman/interpolate v0.0.2 // indirect
	github.com/midtrans/midtrans-go v1.3.8 // indirect
	github.com/pressly/goose/v3 v3.27.1 // indirect
	github.com/sethvargo/go-retry v0.3.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/text v0.36.0 // indirect
)
