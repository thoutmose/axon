package l10n

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// keysInCode are the strings the API writes to people: problem details,
// validation messages, and what goes through T, Tr, or Mark.
func keysInCode(t *testing.T) []string {
	t.Helper()
	seen := map[string]bool{}
	add := func(e ast.Expr) {
		if lit, ok := e.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			if s, err := strconv.Unquote(lit.Value); err == nil {
				seen[s] = true
			}
		}
	}
	fset := token.NewFileSet()
	err := filepath.WalkDir("..", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.CallExpr:
				name := ""
				switch fn := n.Fun.(type) {
				case *ast.SelectorExpr:
					if x, ok := fn.X.(*ast.Ident); ok {
						name = x.Name + "." + fn.Sel.Name
					}
				case *ast.Ident:
					name = f.Name.Name + "." + fn.Name
				}
				switch {
				case name == "httpx.Problem" && len(n.Args) == 3:
					add(n.Args[2])
				case (name == "l10n.T" || name == "l10n.Tr") && len(n.Args) >= 2:
					add(n.Args[1])
				case name == "l10n.Mark" && len(n.Args) == 1:
					add(n.Args[0])
				}
			case *ast.CompositeLit:
				typ := ""
				switch tt := n.Type.(type) {
				case *ast.Ident:
					typ = tt.Name
				case *ast.SelectorExpr:
					typ = tt.Sel.Name
				}
				if typ == "ValidationError" && len(n.Elts) == 1 {
					if kv, ok := n.Elts[0].(*ast.KeyValueExpr); ok {
						add(kv.Value)
					} else {
						add(n.Elts[0])
					}
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func TestEveryMessageHasFrench(t *testing.T) {
	var missing []string
	for _, k := range keysInCode(t) {
		if _, ok := catalog[k]; !ok {
			missing = append(missing, strconv.Quote(k))
		}
	}
	if len(missing) > 0 {
		t.Errorf("%d messages have no French; add them to fr.go:\n%s", len(missing), strings.Join(missing, "\n"))
	}
}

func TestTheCatalogHoldsOnlyUsedMessages(t *testing.T) {
	used := map[string]bool{}
	for _, k := range keysInCode(t) {
		used[k] = true
	}
	for k := range catalog {
		if !used[k] {
			t.Errorf("not used anywhere: %q", k)
		}
	}
}

func TestFrenchKeepsEveryVerb(t *testing.T) {
	verbs := func(s string) []string {
		var out []string
		for i := 0; i < len(s); i++ {
			if s[i] != '%' {
				continue
			}
			j := i + 1
			for j < len(s) && strings.ContainsRune("+-# 0123456789.", rune(s[j])) {
				j++
			}
			if j < len(s) && s[j] != '%' {
				out = append(out, s[i:j+1])
			}
			i = j
		}
		slices.Sort(out)
		return out
	}
	for en, fr := range catalog {
		if !slices.Equal(verbs(en), verbs(fr)) {
			t.Errorf("%q and %q take different arguments", en, fr)
		}
	}
}
