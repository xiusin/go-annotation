package mapper

import (
	"fmt"
	"github.com/xiusin/go-annotation/annotations/mapper/annotations"
	"github.com/xiusin/go-annotation/annotations/mapper/generators"
	cache2 "github.com/xiusin/go-annotation/annotations/mapper/generators/cache"
	"github.com/xiusin/go-annotation/annotations/mapper/templates"
	annotation "github.com/xiusin/go-annotation/pkg"
	"go/ast"
	"path/filepath"
	"strings"
)

func init() {
	p := &Processor{cache: map[key][]mapperData{}, impCache: map[key]*cache2.ImportCache{}}
	annotation.Register[annotations.Mapper](p)
	annotation.RegisterNoop[annotations.Mapping]()
	annotation.RegisterNoop[annotations.SliceMapping]()
	annotation.RegisterNoop[annotations.MapMapping]()
	annotation.RegisterNoop[annotations.IgnoreDefaultMapping]()
}

var _ annotation.AnnotationProcessor = (*Processor)(nil)

type key struct {
	dir string
	pkg string
}

type Processor struct {
	cache    map[key][]mapperData
	impCache map[key]*cache2.ImportCache
}

type mapperData struct {
	data    []byte
	imports []generators.Import
}

func (p *Processor) Process(node annotation.Node) error {
	a, ts, err := validateAndGetMapperWithTypeSpec(node)
	if err != nil {
		return err
	}

	if ts == nil {
		return nil
	}

	mapperName, err := a.BuildName(ts.Name.String())
	if err != nil {
		return fmt.Errorf("unable to build mapper name: %w", err)
	}

	k := key{
		dir: node.Meta().Dir(),
		pkg: node.Meta().PackageName(),
	}

	impCache, ok := p.impCache[k]
	if !ok {
		impCache = cache2.NewImportCache("_imp_%d")
		p.impCache[k] = impCache
	}

	mapperGenerator := generators.NewMapperGeneratorBuilder().
		Node(node).
		ImpCache(impCache).
		IntName(ts.Name.String()).
		IntType(ts.Type.(*ast.InterfaceType)).
		StructName(mapperName).
		Build()

	data, imports, err := mapperGenerator.Generate()
	if err != nil {
		return fmt.Errorf("unable to generate mapper for %s: %w", ts.Name.String(), err)
	}

	p.cache[k] = append(p.cache[k], mapperData{
		data:    data,
		imports: imports,
	})
	return nil
}

func (p *Processor) Output() map[string][]byte {
	out := map[string][]byte{}

	for k, data := range p.cache {
		var rd []byte
		for _, d := range data {
			rd = append(rd, d.data...)
		}

		data := string(rd)

		cachedImports, ok := p.impCache[k]
		var importsSlice []generators.Import
		var aliasReplace map[string]string
		if ok {
			for _, v := range cachedImports.BuildImports() {
				if !strings.Contains(data, v[0]) {
					continue
				}
				importsSlice = append(importsSlice, generators.Import{
					Alias:  v[0],
					Import: v[1],
				})
			}
			aliasReplace = cachedImports.BuildReplaceMap()
		}

		fileData, err := templates.Execute(templates.FileTemplate, map[string]interface{}{
			"PackageName": k.pkg,
			"Data":        string(rd),
			"HasImports":  len(importsSlice) != 0,
			"Imports":     importsSlice,
		})
		if err != nil {
			panic(err)
		}

		if len(aliasReplace) > 0 {
			fileData, err = templates.ExecuteTemplate(string(fileData), aliasReplace)
		}

		/*distinctImports := map[generators.Import]struct{}{}
		for _, d := range data {
			rd = append(rd, d.data...)
			for _, g := range d.imports {
				distinctImports[g] = struct{}{}
			}
		}
		importsSlice := make([]generators.Import, len(distinctImports))
		var ind int
		for imp, _ := range distinctImports {
			importsSlice[ind] = imp
			ind++
		}*/
		if err != nil {
			panic(err)
		}
		out[filepath.Join(k.dir, "mappers.gen.go")] = fileData
	}

	return out
}

func (p *Processor) Version() string {
	return "0.0.1-alpha"
}

func (p *Processor) Name() string {
	return "Mapper"
}
