package gqlidgen

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/99designs/gqlgen/plugin"
)

const outputFileName = "gqlidgen_gen.go"
const configMutatorTemplate = "config_mutator.gotpl"
const codeGeneratorTemplate = "code_generator.gotpl"
const bindingMethodPrefix = "GetGqlIDField_"

func New(modelDir string, modelPackage string) plugin.Plugin {
	return &Plugin{
		outputFilePath: path.Join(modelDir, outputFileName),
		modelDir:       modelDir,
		modelPackage:   modelPackage,
	}
}

type Plugin struct {
	outputFilePath string
	modelDir       string
	modelPackage   string
}

func (m *Plugin) Name() string { return "gqlidgen" }

func (m *Plugin) MutateConfig(cfg *config.Config) error {
	_ = syscall.Unlink(m.outputFilePath)
	implementors := make([]string, 0)
	for typeName, interfaces := range cfg.Schema.Implements {
		for _, i := range interfaces {
			if i.Name == "Node" {
				implementors = append(implementors, typeName)
				break
			}
		}
	}
	if len(implementors) == 0 {
		return nil
	}

	templateStr, err := readTemplateFile(configMutatorTemplate)
	if err != nil {
		return err
	}

	return templates.Render(templates.Options{
		PackageName: m.modelPackage,
		Template:    templateStr,
		Filename:    m.outputFilePath,
		Data: &ConfigMutateTemplateData{
			NodeImplementors: implementors,
		},
		GeneratedHeader: true,
		Packages:        cfg.Packages,
	})
}

func (m *Plugin) GenerateCode(data *codegen.Data) error {
	implementors := getNodeImplementors(data.Objects, m.modelPackage)
	if len(implementors) == 0 {
		return nil
	}

	var addedImports []string
	seenImports := make(map[string]bool)
	for _, i := range implementors {
		for _, p := range i.Packages {
			if p != "" && !seenImports[p] {
				seenImports[p] = true
				addedImports = append(addedImports, p)
			}
		}
	}

	templateStr, err := readTemplateFile(codeGeneratorTemplate)
	if err != nil {
		return err
	}

	return templates.Render(templates.Options{
		PackageName: m.modelPackage,
		Template:    templateStr,
		Filename:    m.outputFilePath,
		Data: &CodegenTemplateData{
			Data:             data,
			AddedImports:     addedImports,
			NodeImplementors: implementors,
		},
		GeneratedHeader: true,
		Packages:        data.Config.Packages,
	})
}

type nodeImplementor struct {
	Types                   []string
	TypeIsPointer           []bool
	Args                    []string
	Packages                []string
	Name                    string
	Implementation          string
	HasBindingMethods       bool
	BindingMethodSignatures []string
}

func getNodeImplementors(objects []*codegen.Object, modelPackage string) []nodeImplementor {
	res := make([]nodeImplementor, 0)
	for _, obj := range objects {
		for _, impl := range obj.Implements {
			if impl.Name == "Node" {
				// Look for ID field - simple and straightforward
				var idField string
				candidates := []string{"ID", obj.Name + "ID", "DBID", "Id"}
				for _, c := range candidates {
					for _, f := range obj.Fields {
						if strings.EqualFold(f.Name, c) {
							idField = f.GoFieldName
							break
						}
					}
					if idField != "" {
						break
					}
				}

				if idField != "" {
					res = append(res, nodeImplementor{
						Name:           obj.Name,
						Implementation: fmt.Sprintf(`GqlID(fmt.Sprintf("%s:%%s", r.%s))`, obj.Name, idField),
						Types:          []string{"persist.DBID"},
						TypeIsPointer:  []bool{false},
						Args:           []string{"id"},
						Packages:       []string{"github.com/mutuals/go-mutuals/service/persist"},
					})
				}
			}
		}
	}
	return res
}

type ConfigMutateTemplateData struct{ NodeImplementors []string }
type CodegenTemplateData struct {
	*codegen.Data
	AddedImports     []string
	NodeImplementors []nodeImplementor
}

func readTemplateFile(filename string) (string, error) {
	_, callerFile, _, _ := runtime.Caller(1)
	rootDir := filepath.Dir(callerFile)
	templatePath := filepath.Join(rootDir, filename)
	data, err := os.ReadFile(templatePath)
	return string(data), err
}
