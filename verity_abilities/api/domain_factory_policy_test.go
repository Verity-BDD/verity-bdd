package api_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const verityImportPath = "github.com/verity-bdd/verity-bdd"

var allowedDomainConstructors = map[string]struct{}{
	verityImportPath + ".NewVerityTest":                                             {},
	verityImportPath + ".NewVerityTestWithContext":                                  {},
	verityImportPath + ".NewVerityTestWithReporter":                                 {},
	verityImportPath + "/verity_abilities/take_notes.NewNoteBook":                   {},
	verityImportPath + "/verity_reporting/allure_reporter.NewAllureReporterWithDir": {},
	verityImportPath + "/verity_reporting/console_reporter.NewConsoleReporter":      {},
	verityImportPath + "/verity_reporting.NewTestRunnerAdapter":                     {},
	verityImportPath + "/verity_reporting.NewActivityTracker":                       {},
	verityImportPath + "/verity_reporting.NewActivityTrackerWithActor":              {},
}

func TestPublicDomainFactoriesUseDomainOrientedNames(t *testing.T) {
	t.Parallel()

	forbidden, err := findConstructorOrientedDomainFactories(repositoryRoot(t))
	if err != nil {
		t.Fatalf("scan public API: %v", err)
	}
	if len(forbidden) > 0 {
		t.Fatalf("constructor-oriented public domain factories: %s", strings.Join(forbidden, ", "))
	}
}

type listedPackage struct {
	Dir        string
	ImportPath string
	GoFiles    []string
	CgoFiles   []string
}

type domainContracts struct {
	activity    *types.Interface
	interaction *types.Interface
	question    *types.Named
}

func findConstructorOrientedDomainFactories(moduleDir string) ([]string, error) {
	packages, err := listPackages(moduleDir)
	if err != nil {
		return nil, err
	}

	fileSet := token.NewFileSet()
	typeImporter := newGoListImporter(moduleDir, fileSet)
	verityPackage, err := typeImporter.Import(verityImportPath)
	if err != nil {
		return nil, fmt.Errorf("import domain contracts: %w", err)
	}
	contracts, err := contractsFrom(verityPackage)
	if err != nil {
		return nil, err
	}

	var forbidden []string
	for _, listed := range packages {
		if excludedPackage(listed.ImportPath) {
			continue
		}
		found, err := scanPackage(fileSet, typeImporter, listed, contracts)
		if err != nil {
			return nil, err
		}
		for _, symbol := range found {
			if _, allowed := allowedDomainConstructors[symbol]; !allowed {
				forbidden = append(forbidden, symbol)
			}
		}
	}
	sort.Strings(forbidden)
	return forbidden, nil
}

func listPackages(moduleDir string) ([]listedPackage, error) {
	command := exec.Command("go", "list", "-json", "./...")
	command.Dir = moduleDir
	output, err := command.Output()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return nil, fmt.Errorf("go list ./...: %s", bytes.TrimSpace(exitError.Stderr))
		}
		return nil, fmt.Errorf("go list ./...: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(output))
	var packages []listedPackage
	for decoder.More() {
		var listed listedPackage
		if err := decoder.Decode(&listed); err != nil {
			return nil, fmt.Errorf("decode go list output: %w", err)
		}
		packages = append(packages, listed)
	}
	return packages, nil
}

func scanPackage(fileSet *token.FileSet, typeImporter types.Importer, listed listedPackage, contracts domainContracts) ([]string, error) {
	filenames := append(append([]string{}, listed.GoFiles...), listed.CgoFiles...)
	files := make([]*ast.File, 0, len(filenames))
	for _, filename := range filenames {
		file, err := parser.ParseFile(fileSet, filepath.Join(listed.Dir, filename), nil, 0)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", filepath.Join(listed.Dir, filename), err)
		}
		files = append(files, file)
	}
	if len(files) == 0 {
		return nil, nil
	}

	info := &types.Info{Defs: make(map[*ast.Ident]types.Object)}
	configuration := types.Config{Importer: typeImporter, IgnoreFuncBodies: true}
	if _, err := configuration.Check(listed.ImportPath, fileSet, files, info); err != nil {
		return nil, fmt.Errorf("type-check %s: %w", listed.ImportPath, err)
	}

	var found []string
	for _, file := range files {
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv != nil || !ast.IsExported(function.Name.Name) || !strings.HasPrefix(function.Name.Name, "New") {
				continue
			}
			object, ok := info.Defs[function.Name].(*types.Func)
			if !ok {
				continue
			}
			if signatureReturnsDomain(object.Type().(*types.Signature), contracts) {
				found = append(found, listed.ImportPath+"."+function.Name.Name)
			}
		}
	}
	return found, nil
}

func signatureReturnsDomain(signature *types.Signature, contracts domainContracts) bool {
	for index := 0; index < signature.Results().Len(); index++ {
		result := signature.Results().At(index).Type()
		if isDomainType(result, contracts) || isFluentDomainBuilder(result, contracts) {
			return true
		}
	}
	return false
}

func isDomainType(candidate types.Type, contracts domainContracts) bool {
	if types.Implements(candidate, contracts.activity) || types.Implements(candidate, contracts.interaction) {
		return true
	}

	method, _, _ := types.LookupFieldOrMethod(candidate, true, nil, "AnsweredBy")
	answeredBy, ok := method.(*types.Func)
	if !ok {
		return false
	}
	signature, ok := answeredBy.Type().(*types.Signature)
	if !ok || signature.Results().Len() != 2 {
		return false
	}
	instantiated, err := types.Instantiate(nil, contracts.question, []types.Type{signature.Results().At(0).Type()}, false)
	if err != nil {
		return false
	}
	question, ok := instantiated.Underlying().(*types.Interface)
	return ok && types.Implements(candidate, question)
}

func isFluentDomainBuilder(candidate types.Type, contracts domainContracts) bool {
	methods := types.NewMethodSet(candidate)
	fluent := false
	terminal := false
	for index := 0; index < methods.Len(); index++ {
		selection := methods.At(index)
		method := selection.Obj()
		if !method.Exported() {
			continue
		}
		signature, ok := method.Type().(*types.Signature)
		if !ok {
			continue
		}
		for resultIndex := 0; resultIndex < signature.Results().Len(); resultIndex++ {
			result := signature.Results().At(resultIndex).Type()
			fluent = fluent || types.Identical(result, candidate)
			terminal = terminal || isDomainType(result, contracts)
		}
	}
	return fluent && terminal
}

func contractsFrom(verityPackage *types.Package) (domainContracts, error) {
	activity, err := interfaceFromScope(verityPackage, "Activity")
	if err != nil {
		return domainContracts{}, err
	}
	interaction, err := interfaceFromScope(verityPackage, "Interaction")
	if err != nil {
		return domainContracts{}, err
	}
	questionObject, ok := verityPackage.Scope().Lookup("Question").(*types.TypeName)
	if !ok {
		return domainContracts{}, fmt.Errorf("domain contract Question is not a named type")
	}
	question, ok := questionObject.Type().(*types.Named)
	if !ok {
		return domainContracts{}, fmt.Errorf("domain contract Question is not generic named type")
	}
	return domainContracts{activity: activity, interaction: interaction, question: question}, nil
}

func interfaceFromScope(pkg *types.Package, name string) (*types.Interface, error) {
	object, ok := pkg.Scope().Lookup(name).(*types.TypeName)
	if !ok {
		return nil, fmt.Errorf("domain contract %s is not a named type", name)
	}
	contract, ok := object.Type().Underlying().(*types.Interface)
	if !ok {
		return nil, fmt.Errorf("domain contract %s is not an interface", name)
	}
	return contract.Complete(), nil
}

func excludedPackage(importPath string) bool {
	for _, component := range strings.Split(importPath, "/") {
		if component == "internal" || component == "mocks" {
			return true
		}
	}
	return false
}

type goListImporter struct {
	dir      string
	compiler types.Importer
}

func newGoListImporter(dir string, fileSet *token.FileSet) *goListImporter {
	result := &goListImporter{dir: dir}
	result.compiler = importer.ForCompiler(fileSet, "gc", result.lookup)
	return result
}

func (current *goListImporter) Import(path string) (*types.Package, error) {
	return current.compiler.Import(path)
}

func (current *goListImporter) lookup(path string) (io.ReadCloser, error) {
	command := exec.Command("go", "list", "-export", "-f={{.Export}}", path)
	command.Dir = current.dir
	output, err := command.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("find export data for %s: %s", path, bytes.TrimSpace(output))
	}
	exportFile := strings.TrimSpace(string(output))
	if exportFile == "" {
		return nil, fmt.Errorf("no export data for %s", path)
	}
	return os.Open(exportFile)
}
