package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// ValidateEncapsulation scans codebase for encapsulation violations
func ValidateEncapsulation(rootDir string, mode string) error {
	violations := []Violation{}

	fmt.Printf("Scanning Go files in %s for encapsulation violations...\n\n", rootDir)

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip test files and vendor
		if !strings.HasSuffix(path, ".go") ||
		   strings.HasSuffix(path, "_test.go") ||
		   strings.Contains(path, "vendor/") ||
		   strings.Contains(path, ".git/") {
			return nil
		}

		// Parse Go file
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, parser.AllErrors|parser.ParseComments)
		if err != nil {
			// Log warning but continue
			fmt.Fprintf(os.Stderr, "Warning: could not parse %s: %v\n", path, err)
			return nil
		}

		// Scan for container.Resolve() calls
		ast.Inspect(node, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			// Check if this is a Resolve method call
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Resolve" {
				return true
			}

			// Extract service name from first argument
			if len(call.Args) == 0 {
				return true
			}

			// Get the service name if it's a string literal
			var serviceName string
			if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
				serviceName = strings.Trim(lit.Value, `"`)
			} else {
				// If not a string literal, we can't analyze statically
				serviceName = "<dynamic>"
			}

			// Get the context line
			position := fset.Position(call.Pos())
			callPos := call.Pos()

			violation := Violation{
				File:     path,
				Line:     position.Line,
				Function: getFunctionName(node, fset, callPos),
				Service:  serviceName,
			}

			violations = append(violations, violation)
			return true
		})

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to scan directory: %w", err)
	}

	// Report violations
	if len(violations) > 0 {
		fmt.Printf("Found %d potential encapsulation violations:\n\n", len(violations))

		// Group by file for cleaner output
		byFile := make(map[string][]Violation)
		for _, v := range violations {
			byFile[v.File] = append(byFile[v.File], v)
		}

		for file, fileViolations := range byFile {
			relPath, _ := filepath.Rel(rootDir, file)
			fmt.Printf("%s:\n", relPath)
			for _, v := range fileViolations {
				fmt.Printf("  Line %d: %s -> Resolve('%s')\n", v.Line, v.Function, v.Service)
			}
			fmt.Println()
		}

		if mode == "strict" {
			return fmt.Errorf("encapsulation violations detected in strict mode")
		}
	} else {
		fmt.Println("✓ No potential encapsulation violations found")
	}

	return nil
}

// Violation represents a potential encapsulation violation
type Violation struct {
	File     string
	Line     int
	Function string
	Service  string
}

// getFunctionName attempts to find the function containing the position
func getFunctionName(node *ast.File, fset *token.FileSet, callPos token.Pos) string {
	var funcName string

	ast.Inspect(node, func(n ast.Node) bool {
		if fd, ok := n.(*ast.FuncDecl); ok {
			if fd.Pos() <= callPos && callPos <= fd.End() {
				if fd.Recv != nil {
					// Method
					if ident, ok := fd.Recv.List[0].Type.(*ast.Ident); ok {
						funcName = ident.Name + "." + fd.Name.Name
					} else {
						funcName = "method:" + fd.Name.Name
					}
				} else {
					// Function
					funcName = fd.Name.Name
				}
				return false
			}
		}
		return true
	})

	if funcName == "" {
		return "unknown"
	}
	return funcName
}

// printUsage prints the usage information
func printUsage() {
	fmt.Printf(`Usage: %s [options] <project-root>

Options:
  -mode string     Validation mode: "warn" (default) or "strict"
                   - warn: Report violations but don't fail
                   - strict: Exit with error if violations found

  -help, -h       Show this help message

Examples:
  %s ./my-project
  %s -mode=strict ./my-project

`, os.Args[0], os.Args[0], os.Args[0])
}

func main() {
	var mode string
	var help bool

	flag.StringVar(&mode, "mode", "warn", "Validation mode: warn or strict")
	flag.BoolVar(&help, "help", false, "Show help")
	flag.BoolVar(&help, "h", false, "Show help")
	flag.Parse()

	if help {
		printUsage()
		os.Exit(0)
	}

	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Error: Missing project root directory\n\n")
		printUsage()
		os.Exit(1)
	}

	rootDir := flag.Arg(0)

	// Validate mode
	if mode != "warn" && mode != "strict" {
		fmt.Fprintf(os.Stderr, "Error: Invalid mode '%s'. Must be 'warn' or 'strict'\n\n", mode)
		printUsage()
		os.Exit(1)
	}

	// Check if directory exists
	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Directory '%s' does not exist\n", rootDir)
		os.Exit(1)
	}

	// Run validation
	if err := ValidateEncapsulation(rootDir, mode); err != nil {
		fmt.Fprintf(os.Stderr, "\nValidation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✓ Validation completed")
}