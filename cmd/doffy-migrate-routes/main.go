package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// RouteInfo represents a detected route
type RouteInfo struct {
	File      string   `json:"file"`
	Line      int      `json:"line"`
	Method    string   `json:"method"`
	Path      string   `json:"path"`
	IsHard    bool     `json:"is_hard"` // true if path starts with /
	Suggested string   `json:"suggested,omitempty"`
	Context   []string `json:"context,omitempty"` // surrounding lines for context
}

// MigrationReport contains the analysis results
type MigrationReport struct {
	ProjectPath   string       `json:"project_path"`
	TotalRoutes   int          `json:"total_routes"`
	HardCoded     int          `json:"hard_coded"`
	Routes        []RouteInfo  `json:"routes"`
	Modules       []ModuleInfo `json:"modules,omitempty"`
}

// ModuleInfo represents a module that should have a prefix
type ModuleInfo struct {
	Name         string   `json:"name"`
	Suggested    string   `json:"suggested"`
	AffectedFiles []string `json:"affected_files"`
}

var (
	path     = flag.String("path", ".", "Project path to analyze")
	verbose  = flag.Bool("verbose", false, "Verbose output")
	fix      = flag.Bool("fix", false, "Automatically fix common issues")
	output   = flag.String("output", "", "Output file for JSON report (default: stdout)")
	help     = flag.Bool("help", false, "Show help")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Doffy Route Migration Tool\n")
		fmt.Fprintf(os.Stderr, "Detects hard-coded route paths and suggests module-based prefixes\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -path ./examples/user-service\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -path ./src -verbose -fix -output migration-report.json\n", os.Args[0])
	}

	flag.Parse()

	if *help {
		flag.Usage()
		return
	}

	report, err := analyzeProject(*path)
	if err != nil {
		log.Fatalf("Error analyzing project: %v", err)
	}

	// Print summary
	fmt.Printf("\nMigration Analysis Report\n")
	fmt.Printf("========================\n")
	fmt.Printf("Project: %s\n", report.ProjectPath)
	fmt.Printf("Total routes found: %d\n", report.TotalRoutes)
	fmt.Printf("Hard-coded routes: %d (%.1f%%)\n",
		report.HardCoded,
		float64(report.HardCoded)/float64(report.TotalRoutes)*100)

	// Group hard-coded routes by file for module suggestions
	moduleSuggestions := suggestModules(report.Routes)
	report.Modules = moduleSuggestions

	if *verbose {
		fmt.Printf("\nDetailed Route Analysis:\n")
		fmt.Printf("========================\n")
		for _, route := range report.Routes {
			if route.IsHard {
				fmt.Printf("🔴 %s:%d - %s %s\n", route.File, route.Line, route.Method, route.Path)
				if route.Suggested != "" {
					fmt.Printf("   → Suggested: %s %s\n", route.Method, route.Suggested)
				}
			} else {
				fmt.Printf("✅ %s:%d - %s %s\n", route.File, route.Line, route.Method, route.Path)
			}
		}
	}

	// Print module suggestions
	if len(moduleSuggestions) > 0 {
		fmt.Printf("\nModule Prefix Suggestions:\n")
		fmt.Printf("==========================\n")
		for _, module := range moduleSuggestions {
			fmt.Printf("Module: %s\n", module.Name)
			fmt.Printf("  Suggested Prefix: %s\n", module.Suggested)
			fmt.Printf("  Affected Files: %v\n", module.AffectedFiles)
			fmt.Printf("\n")
		}
	}

	// Output JSON report
	if *output != "" {
		jsonData, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			log.Fatalf("Error marshaling JSON: %v", err)
		}
		if err := ioutil.WriteFile(*output, jsonData, 0644); err != nil {
			log.Fatalf("Error writing report file: %v", err)
		}
		fmt.Printf("\nJSON report written to: %s\n", *output)
	}

	// Auto-fix if requested
	if *fix {
		fmt.Printf("\nAuto-fix mode is not yet implemented\n")
		fmt.Printf("Please manually update your routes based on the suggestions above\n")
	}

	if report.HardCoded > 0 {
		os.Exit(1) // Exit with error code if hard-coded routes found
	}
}

// analyzeProject scans the project for route definitions
func analyzeProject(projectPath string) (*MigrationReport, error) {
	report := &MigrationReport{
		ProjectPath: projectPath,
		Routes:      []RouteInfo{},
	}

	// Walk through all .go files
	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip vendor, node_modules, and hidden directories
		if strings.Contains(path, "/vendor/") ||
		   strings.Contains(path, "/node_modules/") ||
		   strings.HasPrefix(filepath.Base(path), ".") {
			return nil
		}

		// Only process Go files
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			routes, err := analyzeFile(path)
			if err != nil {
				log.Printf("Error analyzing file %s: %v", path, err)
				return nil // Continue with other files
			}
			report.Routes = append(report.Routes, routes...)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Calculate statistics
	for _, route := range report.Routes {
		if route.IsHard {
			report.HardCoded++
		}
	}
	report.TotalRoutes = len(report.Routes)

	return report, nil
}

// analyzeFile parses a Go file and extracts route information
func analyzeFile(filePath string) ([]RouteInfo, error) {
	var routes []RouteInfo

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// Read file for context extraction
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(content), "\n")

	// Look for route method calls
	ast.Inspect(node, func(n ast.Node) bool {
		callExpr, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		// Check if it's a route method (GET, POST, etc.)
		selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		method := selExpr.Sel.String()
		if !isHTTPMethod(method) {
			return true
		}

		// Extract path from first argument
		if len(callExpr.Args) < 1 {
			return true
		}

		path, isHard, err := extractPathFromArg(callExpr.Args[0], lines, fset.Position(callExpr.Pos()).Line)
		if err != nil {
			log.Printf("Error extracting path in %s: %v", filePath, err)
			return true
		}

		line := fset.Position(callExpr.Pos()).Line

		// Get context lines
		context := getContextLines(lines, line, 3)

		route := RouteInfo{
			File:    filepath.Base(filePath),
			Line:    line,
			Method:  method,
			Path:    path,
			IsHard:  isHard,
			Context: context,
		}

		// Suggest relative path if hard-coded
		if isHard {
			route.Suggested = suggestRelativePath(path)
		}

		routes = append(routes, route)
		return true
	})

	return routes, nil
}

// isHTTPMethod checks if the method name is a valid HTTP method
func isHTTPMethod(method string) bool {
	methods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "PATCH": true,
		"DELETE": true, "OPTIONS": true, "HEAD": true, "ANY": true,
	}
	return methods[strings.ToUpper(method)]
}

// extractPathFromArg extracts the path string from a function argument
func extractPathFromArg(arg ast.Expr, lines []string, lineNum int) (string, bool, error) {
	basicLit, ok := arg.(*ast.BasicLit)
	if !ok {
		return "", false, fmt.Errorf("argument is not a string literal")
	}

	if basicLit.Kind != token.STRING {
		return "", false, fmt.Errorf("argument is not a string")
	}

	path := strings.Trim(basicLit.Value, `"`)
	isHard := strings.HasPrefix(path, "/")

	return path, isHard, nil
}

// getContextLines returns lines around the target line for context
func getContextLines(lines []string, targetLine int, contextSize int) []string {
	start := targetLine - contextSize - 1
	if start < 0 {
		start = 0
	}

	end := targetLine + contextSize
	if end > len(lines) {
		end = len(lines)
	}

	var context []string
	for i := start; i < end; i++ {
		if i >= 0 && i < len(lines) {
			context = append(context, fmt.Sprintf("%d: %s", i+1, strings.TrimSpace(lines[i])))
		}
	}

	return context
}

// suggestRelativePath converts a hard-coded absolute path to a relative path
func suggestRelativePath(absPath string) string {
	// Remove leading slash and convert to kebab-case if needed
	relPath := strings.TrimPrefix(absPath, "/")

	// Convert camelCase to kebab-case for common patterns
	relPath = regexp.MustCompile(`([a-z])([A-Z])`).ReplaceAllString(relPath, `${1}-${2}`)
	relPath = strings.ToLower(relPath)

	return relPath
}

// suggestModules analyzes routes and suggests module groupings
func suggestModules(routes []RouteInfo) []ModuleInfo {
	fileGroups := make(map[string][]RouteInfo)

	// Group routes by file
	for _, route := range routes {
		if route.IsHard {
			fileGroups[route.File] = append(fileGroups[route.File], route)
		}
	}

	var modules []ModuleInfo

	// Analyze each file group
	for filename, fileRoutes := range fileGroups {
		module := ModuleInfo{
			Name:          extractModuleName(filename),
			AffectedFiles: []string{filename},
		}

		// Extract common prefix from routes
		prefix := extractCommonPrefix(fileRoutes)
		if prefix != "" {
			module.Suggested = "/" + prefix
		} else {
			module.Suggested = "/" + module.Name
		}

		modules = append(modules, module)
	}

	return modules
}

// extractModuleName derives a module name from filename
func extractModuleName(filename string) string {
	base := filepath.Base(filename)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	// Remove common suffixes
	name = strings.TrimSuffix(name, "_test")
	name = strings.TrimSuffix(name, "_routes")
	name = strings.TrimSuffix(name, "_handler")
	name = strings.TrimSuffix(name, "_controller")

	// Convert to kebab-case
	name = regexp.MustCompile(`([a-z])([A-Z])`).ReplaceAllString(name, `${1}-${2}`)
	name = strings.ToLower(name)

	return name
}

// extractCommonPrefix finds common prefix from route paths
func extractCommonPrefix(routes []RouteInfo) string {
	if len(routes) == 0 {
		return ""
	}

	// Get first segment of each path
	var segments []string
	for _, route := range routes {
		path := strings.TrimPrefix(route.Path, "/")
		if path == "" {
			continue
		}

		segments = strings.Split(path, "/")
		if len(segments) > 0 {
			segments = []string{segments[0]}
			break
		}
	}

	if len(segments) == 0 {
		return ""
	}

	// Return the most common first segment
	counts := make(map[string]int)
	for _, route := range routes {
		path := strings.TrimPrefix(route.Path, "/")
		if path == "" {
			continue
		}

		segments := strings.Split(path, "/")
		if len(segments) > 0 {
			counts[segments[0]]++
		}
	}

	var maxSegment string
	maxCount := 0
	for segment, count := range counts {
		if count > maxCount {
			maxCount = count
			maxSegment = segment
		}
	}

	// Only return if it appears in most routes
	if maxCount >= len(routes)/2 {
		return maxSegment
	}

	return ""
}