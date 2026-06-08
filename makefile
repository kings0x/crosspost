#put the commands for checking and installing gh cli
#then creating environments in the repo provided
#verify the exist
#put the infisical secretes and paas platform secrets

#the docker login and push dummy image to ghcr
#infisical install
#infiscal login
#infisical commands

migrate:
	@infisical run --env=dev --path=/ -- go run ./cmd/migrate/main.go up

run:
	@infisical run --env=dev --path=/ -- go run ./cmd/api/main.go

dev: migrate run