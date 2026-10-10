package connectors

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// Every message a connector can show (status details, errors) names where a
// v2 credential really comes from: the process env, ~/.founderos/.env, or a
// key planted under API keys. Never v1's .env.local, another tool's files or
// a private machine path, and never the private build's "bridge".
func TestConnectorMessagesNameOnlyV2CredentialSources(t *testing.T) {
	banned := regexp.MustCompile(`\.?env\.local|clue-agent|social-media/\.env|arcads-agent-skills|~/Projects|\.founderos-bridge|tokens-mcp\.json|\bthe bridge\b`)
	fset := token.NewFileSet()
	err := filepath.WalkDir(".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return err
		}
		f, err := parser.ParseFile(fset, p, nil, 0) // comments dropped: only code strings
		if err != nil {
			return err
		}
		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			s, _ := strconv.Unquote(lit.Value)
			if m := banned.FindString(s); m != "" {
				t.Errorf("%s: %q mentions %q", fset.Position(lit.Pos()), s, m)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSetKeysHintNamesTheV2Sources(t *testing.T) {
	if got := SetKeys("STRIPE_SECRET_KEY"); got != "Set STRIPE_SECRET_KEY in ~/.founderos/.env or under API keys." {
		t.Errorf("one key: %q", got)
	}
	if got := SetKeys("A", "B", "C"); got != "Set A, B and C in ~/.founderos/.env or under API keys." {
		t.Errorf("three keys: %q", got)
	}
	if got := NotFound("MIRO_ACCESS_TOKEN"); got != "MIRO_ACCESS_TOKEN not found in env or ~/.founderos/.env." {
		t.Errorf("not found: %q", got)
	}
}
