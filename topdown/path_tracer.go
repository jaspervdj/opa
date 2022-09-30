package topdown

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/ast/location"
)

type DummyExecutionTracer struct {
	pathSet *PathSet
}

func NewDummyExecutionTracer() *DummyExecutionTracer {
	return &DummyExecutionTracer{
		pathSet: NewPathSet(),
	}
}

func (t *DummyExecutionTracer) term(term *ast.Term) {
	if term != nil && term.IsGround() {
		if term.Location != nil {
			if encoded := t.decodePath(term.Location.File); encoded != nil {
				t.pathSet.Add(encoded)
			}
		}
	}
}

func (t *DummyExecutionTracer) Unify(l *ast.Term, r *ast.Term) {
	fmt.Fprintf(os.Stderr, "DummyExecutionTracer: %s = %s\n", l.String(), r.String())
	t.term(l)
	t.term(r)
}

func (t *DummyExecutionTracer) Builtin(f *ast.Builtin, args []*ast.Term) {
	strs := make([]string, len(args))
	for i, arg := range args {
		strs[i] = fmt.Sprintf("%v", arg)
	}
	fmt.Fprintf(os.Stderr, "DummyExecutionTracer: %s(%s)\n", f.Name, strings.Join(strs, ", "))
	for _, arg := range args {
		t.term(arg)
	}
}

func (t *DummyExecutionTracer) DecorateValue(top ast.Value) error {
	path := []interface{}{}
	var decorateValue func(ast.Value) error
	var decorateTerm func(*ast.Term) error
	decorateTerm = func(term *ast.Term) error {
		encoded, err := t.encodePath(path)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Decorated with %s\n", encoded)
		term.Location = &location.Location{File: encoded}
		return decorateValue(term.Value)
	}
	decorateValue = func(value ast.Value) error {
		switch value := value.(type) {
		case *ast.Array:
			for i := 0; i < value.Len(); i++ {
				path = append(path, i)
				if err := decorateTerm(value.Elem(i)); err != nil {
					return err
				}
				path = path[:len(path)-1]
			}
		case ast.Object:
			for _, key := range value.Keys() {
				if str, ok := key.Value.(ast.String); ok {
					path = append(path, str)
					if err := decorateTerm(value.Get(key)); err != nil {
						return err
					}
					path = path[:len(path)-1]
				}
			}
		}
		return nil
	}
	return decorateValue(top)
}

func (t *DummyExecutionTracer) Covered() [][]interface{} {
	return t.pathSet.List()
}

func (t *DummyExecutionTracer) encodePath(path []interface{}) (string, error) {
	bytes, err := json.Marshal(path)
	if err != nil {
		return "", err
	}
	return "path:" + string(bytes), nil
}

func (t *DummyExecutionTracer) decodePath(encoded string) []interface{} {
	if !strings.HasPrefix(encoded, "path:") {
		return nil
	}
	encoded = strings.TrimPrefix(encoded, "path:")
	var path []interface{}
	if err := json.Unmarshal([]byte(encoded), &path); err != nil {
		return nil
	}
	return path
}

type PathSet struct {
	intKeys    map[int]*PathSet
	stringKeys map[string]*PathSet
}

func NewPathSet() *PathSet {
	return &PathSet{
		intKeys:    map[int]*PathSet{},
		stringKeys: map[string]*PathSet{},
	}
}

func (ps *PathSet) addInt(k int, path []interface{}) {
	if _, ok := ps.intKeys[k]; !ok {
		ps.intKeys[k] = NewPathSet()
	}
	ps.intKeys[k].Add(path)
}

func (ps *PathSet) addString(k string, path []interface{}) {
	if _, ok := ps.stringKeys[k]; !ok {
		ps.stringKeys[k] = NewPathSet()
	}
	ps.stringKeys[k].Add(path)
}

func (ps *PathSet) Add(path []interface{}) {
	if len(path) <= 0 {
		return
	}
	switch k := path[0].(type) {
	case int:
		ps.addInt(k, path[1:])
	case float64:
		ps.addInt(int(k), path[1:])
	case string:
		ps.addString(k, path[1:])
	default:
		fmt.Fprintf(os.Stderr, "uncovered: %t\n", k)
	}
}

func (ps *PathSet) List() [][]interface{} {
	paths := [][]interface{}{}
	for i, child := range ps.intKeys {
		for _, childPath := range child.List() {
			path := make([]interface{}, len(childPath)+1)
			path[0] = i
			copy(path[1:], childPath)
			paths = append(paths, path)
		}
	}
	for i, child := range ps.stringKeys {
		for _, childPath := range child.List() {
			path := make([]interface{}, len(childPath)+1)
			path[0] = i
			copy(path[1:], childPath)
			paths = append(paths, path)
		}
	}
	if len(paths) <= 0 {
		paths = append(paths, []interface{}{})
	}
	return paths
}
