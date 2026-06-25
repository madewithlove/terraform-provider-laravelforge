default: testacc

# Pinned versions of the HashiCorp code generators. Bump deliberately.
TFPLUGINGEN_OPENAPI_VERSION   ?= v0.3.0
TFPLUGINGEN_FRAMEWORK_VERSION ?= v0.4.1

# Source of truth for the Forge API.
OPENAPI_URL  ?= https://forge.laravel.com/api/docs.openapi
RAW_SPEC     := generator/docs.openapi.json
CLEAN_SPEC   := generator/docs.openapi.processed.json
CODE_SPEC    := provider_code_spec.json
GEN_CONFIG   := generator/generator_config.yml
PROVIDER_DIR := internal/provider

# Install the pinned generator tools into $(go env GOBIN) / $(go env GOPATH)/bin.
.PHONY: tools
tools:
	go install github.com/hashicorp/terraform-plugin-codegen-openapi/cmd/tfplugingen-openapi@$(TFPLUGINGEN_OPENAPI_VERSION)
	go install github.com/hashicorp/terraform-plugin-codegen-framework/cmd/tfplugingen-framework@$(TFPLUGINGEN_FRAMEWORK_VERSION)

# Download the latest raw OpenAPI document from Forge.
.PHONY: fetch-spec
fetch-spec:
	curl -fsSL $(OPENAPI_URL) -o $(RAW_SPEC)
	@echo "fetched $(OPENAPI_URL) -> $(RAW_SPEC)"

# Regenerate the provider from the OpenAPI document. This:
#   1. preprocesses the spec (unwrap JSON:API envelopes, collapse identifiers),
#   2. produces the provider code specification (tfplugingen-openapi),
#   3. generates the resource schemas (tfplugingen-framework),
#   4. scaffolds any missing resource implementations (idempotent),
#   5. regenerates the registry documentation.
# Only the generated *_resource_gen.go files are removed up front; hand-written
# resource implementations and provider.go are preserved.
.PHONY: generate
generate:
	go run ./generator/preprocess -in $(RAW_SPEC) -out $(CLEAN_SPEC)
	tfplugingen-openapi generate --config $(GEN_CONFIG) --output $(CODE_SPEC) $(CLEAN_SPEC)
	find $(PROVIDER_DIR) -name '*_resource_gen.go' -delete
	tfplugingen-framework generate resources --input $(CODE_SPEC) --output $(PROVIDER_DIR)
	tfplugingen-framework generate provider --input $(CODE_SPEC) --output $(PROVIDER_DIR)
	go run ./generator/scaffold -dir $(PROVIDER_DIR)
	gofmt -w $(PROVIDER_DIR)
	go generate ./...

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m
