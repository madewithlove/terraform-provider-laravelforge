// Command scaffold bootstraps a hand-editable resource implementation next to
// each generated schema.
//
// tfplugingen-framework only generates the schema and model for a resource
// (e.g. ServersResourceSchema / ServersModel); it does not generate the
// resource.Resource implementation that wires the schema into the provider and
// performs CRUD against the API. This tool writes that implementation once per
// resource package -- delegating Schema() to the generated schema and leaving
// the CRUD methods as clearly marked TODO stubs.
//
// It is idempotent and never clobbers existing work: if a resource file already
// exists for a package (for example because a human has filled in the CRUD
// logic), that package is skipped. Re-running after generating new resources
// only scaffolds the new ones.
//
// Usage:
//
//	go run ./generator/scaffold -dir internal/provider
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

// schemaFuncRE captures the exported prefix of the generated schema function,
// e.g. "Servers" from "func ServersResourceSchema(ctx".
var schemaFuncRE = regexp.MustCompile(`func ([A-Za-z0-9]+)ResourceSchema\(ctx`)

type resourceInfo struct {
	Package  string // e.g. resource_servers
	Prefix   string // e.g. Servers
	TypeName string // e.g. servers (terraform type suffix)
}

const resourceTemplate = `package {{ .Package }}

// This file was bootstrapped by ./generator/scaffold. Unlike *_resource_gen.go
// it is NOT regenerated, so it is safe to implement the CRUD methods below.

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// Ensure the implementation satisfies the resource.Resource interface.
var _ resource.Resource = (*{{ .Unexported }}Resource)(nil)

// New{{ .Prefix }}Resource returns a new {{ .TypeName }} resource.
func New{{ .Prefix }}Resource() resource.Resource {
	return &{{ .Unexported }}Resource{}
}

type {{ .Unexported }}Resource struct{}

func (r *{{ .Unexported }}Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_{{ .TypeName }}"
}

func (r *{{ .Unexported }}Resource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = {{ .Prefix }}ResourceSchema(ctx)
}

func (r *{{ .Unexported }}Resource) Create(_ context.Context, _ resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError("Not implemented", notImplementedMessage)
}

func (r *{{ .Unexported }}Resource) Read(_ context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) {
	resp.Diagnostics.AddError("Not implemented", notImplementedMessage)
}

func (r *{{ .Unexported }}Resource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Not implemented", notImplementedMessage)
}

func (r *{{ .Unexported }}Resource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError("Not implemented", notImplementedMessage)
}

const notImplementedMessage = "The {{ .TypeName }} resource exposes its generated schema but does not yet implement CRUD against the Laravel Forge API."
`

func main() {
	dir := flag.String("dir", "internal/provider", "directory containing the generated resource_* packages")
	flag.Parse()

	entries, err := os.ReadDir(*dir)
	if err != nil {
		log.Fatalf("reading %s: %v", *dir, err)
	}

	tmpl := template.Must(template.New("resource").Parse(resourceTemplate))

	created, skipped := 0, 0
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "resource_") {
			continue
		}
		pkgDir := filepath.Join(*dir, entry.Name())
		typeName := strings.TrimPrefix(entry.Name(), "resource_")

		// The hand-written implementation lives next to the generated schema.
		implPath := filepath.Join(pkgDir, typeName+"_resource.go")
		if _, err := os.Stat(implPath); err == nil {
			skipped++
			continue
		}

		prefix, err := schemaPrefix(pkgDir)
		if err != nil {
			log.Fatalf("%s: %v", entry.Name(), err)
		}

		info := struct {
			resourceInfo
			Unexported string
		}{
			resourceInfo: resourceInfo{Package: entry.Name(), Prefix: prefix, TypeName: typeName},
			Unexported:   strings.ToLower(prefix[:1]) + prefix[1:],
		}

		f, err := os.Create(implPath)
		if err != nil {
			log.Fatalf("creating %s: %v", implPath, err)
		}
		if err := tmpl.Execute(f, info); err != nil {
			f.Close()
			log.Fatalf("writing %s: %v", implPath, err)
		}
		f.Close()
		created++
	}

	fmt.Printf("scaffolded %d resource(s), skipped %d existing\n", created, skipped)
}

// schemaPrefix finds the exported prefix of the generated schema function in a
// resource package directory.
func schemaPrefix(pkgDir string) (string, error) {
	matches, _ := filepath.Glob(filepath.Join(pkgDir, "*_resource_gen.go"))
	for _, m := range matches {
		b, err := os.ReadFile(m)
		if err != nil {
			return "", err
		}
		if sub := schemaFuncRE.FindSubmatch(b); sub != nil {
			return string(sub[1]), nil
		}
	}
	return "", fmt.Errorf("no *ResourceSchema function found in %s", pkgDir)
}
