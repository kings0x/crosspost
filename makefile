#put the commands for checking and installing gh cli
#then creating environments in the repo provided
#verify the exist
#put the infisical secretes and paas platform secrets

#the docker login and push dummy image to ghcr


.PHONY: gh-setup gh-envs gh-verify gh-secrets gh-staging gh-production help

REPO := kings0x/crosspost
STAGING_ENV := staging
PRODUCTION_ENV := production

# ────────────────────────────────────────────
# GitHub Environments
# ────────────────────────────────────────────

gh-envs:
	@echo "Creating environments..."
	@gh api repos/kings0x/crossPost/environments/staging -X PUT > /dev/null \
		&& echo "✓ staging created"
	@gh api repos/kings0x/crossPost/environments/production -X PUT > /dev/null \
		&& echo "✓ production created"

gh-verify:
	@echo "Environments:"
	@gh api repos/kings0x/crossPost/environments | jq '.environments[] | .name'
	@echo ""
	@echo "Staging secrets:"
	@gh secret list --env staging
	@echo ""
	@echo "Production secrets:"
	@gh secret list --env production

# ────────────────────────────────────────────
# Secrets
# ────────────────────────────────────────────

gh-staging:
	@[ -n "$(CLIENT_ID)" ] || { echo "✗ CLIENT_ID is required"; exit 1; }
	@[ -n "$(CLIENT_SECRET)" ] || { echo "✗ CLIENT_SECRET is required"; exit 1; }
	@[ -n "$(PROJECT_ID)" ] || { echo "✗ PROJECT_ID is required"; exit 1; }
	@echo "Pushing secrets to staging..."
	@gh secret set INFISICAL_CLIENT_ID \
		--env staging --body "$(CLIENT_ID)" && echo "✓ INFISICAL_CLIENT_ID"
	@gh secret set INFISICAL_CLIENT_SECRET \
		--env staging --body "$(CLIENT_SECRET)" && echo "✓ INFISICAL_CLIENT_SECRET"
	@gh secret set INFISICAL_PROJECT_ID \
		--env staging --body "$(PROJECT_ID)" && echo "✓ INFISICAL_PROJECT_ID"

gh-production:
	@[ -n "$(CLIENT_ID)" ] || { echo "✗ CLIENT_ID is required"; exit 1; }
	@[ -n "$(CLIENT_SECRET)" ] || { echo "✗ CLIENT_SECRET is required"; exit 1; }
	@[ -n "$(PROJECT_ID)" ] || { echo "✗ PROJECT_ID is required"; exit 1; }
	@echo "Pushing secrets to production..."
	@gh secret set INFISICAL_CLIENT_ID \
		--env production --body "$(CLIENT_ID)" && echo "✓ INFISICAL_CLIENT_ID"
	@gh secret set INFISICAL_CLIENT_SECRET \
		--env production --body "$(CLIENT_SECRET)" && echo "✓ INFISICAL_CLIENT_SECRET"
	@gh secret set INFISICAL_PROJECT_ID \
		--env production --body "$(PROJECT_ID)" && echo "✓ INFISICAL_PROJECT_ID"

gh-secrets: gh-staging gh-production

gh-setup: gh-envs gh-secrets
	@echo ""
	@echo "Running verification..."
	@$(MAKE) gh-verify

# ────────────────────────────────────────────
# Global secrets (same value across all environments)
# ────────────────────────────────────────────

gh-global:
	@[ -n "$(CLIENT_ID)" ] || { echo "✗ CLIENT_ID is required"; exit 1; }
	@[ -n "$(CLIENT_SECRET)" ] || { echo "✗ CLIENT_SECRET is required"; exit 1; }
	@[ -n "$(PROJECT_ID)" ] || { echo "✗ PROJECT_ID is required"; exit 1; }
	@echo "Pushing global secrets..."
	@gh secret set INFISICAL_CLIENT_ID \
		--body "$(CLIENT_ID)" && echo "✓ INFISICAL_CLIENT_ID"
	@gh secret set INFISICAL_CLIENT_SECRET \
		--body "$(CLIENT_SECRET)" && echo "✓ INFISICAL_CLIENT_SECRET"
	@gh secret set INFISICAL_PROJECT_ID \
		--body "$(PROJECT_ID)" && echo "✓ INFISICAL_PROJECT_ID"

gh-verify-global:
	@echo "Global secrets:"
	@gh secret list

# ────────────────────────────────────────────
# Help
# ────────────────────────────────────────────

help:
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@echo "  gh-setup       create envs + push all secrets"
	@echo "  gh-envs        create staging and production environments"
	@echo "  gh-verify      verify environments and secrets exist"
	@echo "  gh-staging     push secrets to staging"
	@echo "  gh-production  push secrets to production"
	@echo "  gh-secrets     push secrets to both environments"
	@echo ""
	@echo "  Secrets targets require arguments:"
	@echo "  make gh-staging CLIENT_ID=xxx CLIENT_SECRET=xxx PROJECT_ID=xxx"
	@echo ""

migrate:
	@infisical run --env=dev --path=/ -- go run ./cmd/migrate/main.go up

run:
	@infisical run --env=dev --path=/ -- go run ./cmd/api/main.go

dev: migrate run

