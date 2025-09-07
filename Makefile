# Include env variablesmakefil
include .envrc

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^//'

.PHONY: confirm
confirm:
	@echo 'Are you sure? [y/N]' && read ans && [ $${ans:-N} = y ]

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## run/api: run the cmd/api application
.PHONY: run/api
run/api:
	@go run ./cmd/api -db-dsn=${GREENLIGHT_DB_DSN}


## test/setup: runs setup scripts for creating docker container
.PHONY: test/setup
test/setup:
	./cmd/tests/book_setup.sh

## test/data: test internal data
.PHONY: test/data
test/data:
	@go test ./internal/data -v -dsn=${GREENLIGHT_DB_DSN} -open-ai-key=${OPENAIKEY} 

## db/psql: connect to the database using psql
.PHONY: db/psql
db/psql:
	psql ${GREENLIGHT_DB_DSN}
	#psql postgres://postgres:postgres@localhost:5432/testdb?sslmode=disable
	# PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -d testdb
	#

## db/show/subtypes: connect to the database using psql
.PHONY: db/show/subtypes
db/show/subtypes:
	psql ${GREENLIGHT_DB_DSN} -c "SELECT * FROM subtypes;"


## db/migrations/new name=$1: create a new database migration
.PHONY: db/migrations/new
db/migrations/new:
	@echo 'Creating migration files for ${name}'
	migrate create -seq -ext=.sql -dir=./migrations ${name}

## db/migrations/up: apply all up database migrations
.PHONY: db/migrations/up
db/migrations/up: confirm
	@echo 'Running up migrations...'
	migrate -path="./migrations" -database "postgres://greenlight:${DB_PW}@localhost/greenlight?sslmode=disable" up

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## audit: tidy dependencies and format, vet, and test all code
.PHONY: audit
audit: vendor
	@echo 'Formatting code...'
	go fmt ./...
	@echo 'Vetting code...'
	go vet ./...
	staticcheck ./...
	@echo 'Running tests...'
	go test -race -vet=off ./...

## vendor: tidy and vendor dependencies
.PHONY: vendor
vendor:
	@echo 'Tidying and verifying module dependencies'
	go mod tidy
	go mod verify
	@echo 'Vendoring dependencies'
	go mod vendor

# ==================================================================================== #
# BUILD
# ==================================================================================== #

## build/api: build the cmd/api application
.PHONY: build/api
build/api:
	@echo 'Building cmd/api...'
	go build -ldflags '-s -w' -o ./bin/api ./cmd/api
	GOOS=linux GOARCH=amd64 go build -ldflags='-s -w' -o=./bin/linux_amd64/api ./cmd/api


# ==================================================================================== #
# PRODUCTION
# ==================================================================================== #

production_host_ip = '144.126.210.226'

## production/connect: connect to the production server
.PHONY: production/connect
production/connect:
	kubectl exec -it ${POD_NAME} -- psql -h localhost -U ${PSQL_USERNAME} -p 5432 ${DB_NAME}

## production/database/install deploy the database to production
.PHONY: production/database/install
production/database/install:
	helm install typedex ./charts/postgres/

## production/database/uninstall deploy the database to production
.PHONY: production/database/uninstall
production/database/uninstall:
	helm uninstall typedex ./charts/postgres/

## production/api/repo_deploy: deploy the database to production from repot
.PHONY: production/api/repo_deploy
production/api/repo_deploy:
	oc project ${PROJECT_NAME}
	oc start-build ${API_NAME}
