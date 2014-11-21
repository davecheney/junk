package main

import (
	"encoding/json"
	"fmt"
	"go/build"
	"log"
	"math/rand"
	"net/http"
)

type Node struct {
	Id    string  `json:"id"`
	Label string  `json:"label,omitempty"`
	X     float64 `json:"x,omitempty"`
	Y     float64 `json:"y,omitempty"`
	Size  float64 `json:"size"`
}

type Edge struct {
	Id     string `json:"id,omitempty"`
	Source string `json:"source"`
	Target string `json:"target"`
}

func findImport(pkgs map[string][]Node, p string, size float64) {
	if p == "C" {
		return
	}
	//n := Node{Id:p, Label:p}
	if _, ok := pkgs[p]; ok {
		// seen this package before, skip it
		return
	}
	pkg, err := build.Import(p, "", 0)
	if err != nil {
		log.Fatal(err)
	}
	filter := func(imports []string) []Node {
		var n []Node
		for _, p := range imports {
			n = append(n, Node{Id: p, Label: p})
		}
		return n
	}
	pkgs[p] = filter(pkg.Imports)
	for _, pkg := range pkgs[p] {
		findImport(pkgs, pkg.Id, size/2)
	}
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	http.HandleFunc("/data/", data)
	http.HandleFunc("/links/", links)
	http.HandleFunc("/imports/", imports)
	http.HandleFunc("/csv/", csv)
	http.HandleFunc("/csvimports/", csvimports)
	http.HandleFunc("/cc/", cc)
	http.HandleFunc("/pushdown/", pushdown)

	for i := range visuals {
		visual := visuals[i]
		http.HandleFunc("/"+visual.name+"/", func(w http.ResponseWriter, r *http.Request) {
			pkg := r.URL.Path[len("/"+visual.name+"/"):]
			visual.tmpl.Execute(w, map[string]string{"package": pkg})
		})
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		pkg := r.URL.Path[1:] // strip leading /
		index.Execute(w, map[string]string{"package": pkg})
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func imports(w http.ResponseWriter, r *http.Request) {
	pkg := r.URL.Path[len("/imports/"):] // strip leading /data/
	root := walk(pkg)

	enc := json.NewEncoder(w)
	enc.Encode(children(root))
}

func cc(w http.ResponseWriter, r *http.Request) {
	pkg := r.URL.Path[len("/cc/"):] // strip leading /data/
	root := walk(pkg)
	type Node struct {
		Name    string   `json:"name"`
		Imports []string `json:"imports,omitempty"`
	}
	var nodes []Node
	imports := func(p *Package) []string {
		var s []string
		for _, c := range p.Children {
			s = append(s, c.Name)
		}
		return s
	}
	for _, p := range flatten(root) {
		nodes = append(nodes, Node{Name: p.Name, Imports: imports(p)})
	}
	enc := json.NewEncoder(w)
	enc.Encode(nodes)
}

func children(root *Package) interface{} {
	type node struct {
		Name     string `json:"name"`
		Children []node `json:"children,omitempty"`
	}
	var f func(*Package) node
	f = func(root *Package) node {
		var ch []node
		for _, c := range root.Children {
			ch = append(ch, f(c))
		}
		return node{Name: root.Name, Children: ch}
	}
	return f(root)
}

func data(w http.ResponseWriter, r *http.Request) {
	pkg := r.URL.Path[len("/data/"):] // strip leading /data/
	pkgs := make(map[string][]Node)
	findImport(pkgs, pkg, 1.0)

	keys := make(map[string]bool)
	for k, v := range pkgs {
		keys[k] = true
		for _, v := range v {
			keys[v.Id] = true
		}
	}
	var nodes []Node
	for k := range keys {
		nodes = append(nodes, Node{
			Id:    k,
			Label: k,
			X:     rand.Float64(),
			Y:     rand.Float64(),
			Size:  1,
		})
	}

	var edges []Edge
	for k, v := range pkgs {
		for _, p := range v {
			edges = append(edges, Edge{
				Id:     p.Id + "-" + k,
				Source: p.Id,
				Target: k,
			})
		}
	}

	enc := json.NewEncoder(w)
	enc.Encode(struct {
		Nodes []Node `json:"nodes"`
		Edges []Edge `json:"edges"`
	}{Nodes: nodes, Edges: edges})
}

func links(w http.ResponseWriter, r *http.Request) {
	pkg := r.URL.Path[len("/links/"):]
	root := walk(pkg)
	type Node struct {
		Name  string `json:"name"`
		Group int    `json:"group"`
	}
	var nodes []Node
	var f func(*Package, int)
	seen := make(map[*Package]bool)
	targets := make(map[*Package]int) // maps from package to index in pkgs
	var t int
	f = func(p *Package, group int) {
		if !seen[p] {
			nodes = append(nodes, Node{Name: p.Name, Group: group})
			seen[p] = true
			targets[p] = t
			t++
		}
		group++
		for _, c := range p.Children {
			f(c, group)
		}
	}
	f(root, 1)
	type Link struct {
		Source int `json:"source"`
		Target int `json:"target"`
	}
	var links []Link
	pkgs := flatten(root)
	for _, source := range pkgs {
		for _, target := range source.Children {
			links = append(links, Link{Source: targets[source], Target: targets[target]})
		}
	}
	enc := json.NewEncoder(w)
	enc.Encode(struct {
		Nodes []Node `json:"nodes"`
		Links []Link `json:"links"`
	}{Nodes: nodes, Links: links})
}

func csv(w http.ResponseWriter, r *http.Request) {
	pkg := r.URL.Path[len("/csv/"):]
	root := walk(pkg)
	fmt.Fprintln(w, "source,target,weight")
	for _, source := range flatten(root) {
		for _, target := range source.Children {
			fmt.Fprintf(w, "%s,%s,%v\n", source.Name, target.Name, 1)
		}
	}
}

func csvimports(w http.ResponseWriter, r *http.Request) {
	pkg := r.URL.Path[len("/csvimports/"):]
	root := walk(pkg)
	nodes := flatten(root)
	weights := make(map[*Package]float64)
	for _, source := range nodes {
		for _, target := range source.Children {
			weights[target] += 1
		}
	}
	fmt.Fprintln(w, "has,prefers,count")

	for _, source := range nodes {
		for _, target := range source.Children {
			fmt.Fprintf(w, "%s,%s,%v\n", source.Name, target.Name, weights[target])
		}
	}
}

func flatten(root *Package) []*Package {
	seen := make(map[*Package]bool)
	var pkgs []*Package
	var f func(*Package)
	f = func(p *Package) {
		if seen[p] {
			return
		}
		seen[p] = true
		pkgs = append(pkgs, p)
		for _, ch := range p.Children {
			f(ch)
		}
	}
	f(root)
	return pkgs
}

type Package struct {
	Name     string
	Parent   *Package
	Children []*Package
}

func (p Package) String() string {
	return p.Name
}

// walk resolves the package import graph from p to its roots.
func walk(p string) *Package {
	pkgs := make(map[string]*Package)
	var f func(*Package, string) *Package
	f = func(parent *Package, p string) *Package {
		if pk, found := pkgs[p]; found {
			return pk
		}
		pk := Package{
			Name:   p,
			Parent: parent,
		}
		switch p {
		case "C", "unsafe", "runtime":
			// don't resolve dependencies, these
			// packages don't have any, or don't exist.
		default:
			pkg, err := build.Import(p, "", 0)
			if err != nil {
				log.Fatal(err)
			}
			for _, i := range pkg.Imports {
				pk.Children = append(pk.Children, f(&pk, i))
			}
		}
		pkgs[p] = &pk
		return &pk
	}
	return f(nil, p)
}

func tree(p *Package) string {
	if p == nil {
		return ""
	}
	return tree(p.Parent) + "->" + p.String()
}

// explode makes a deep copy of p, expanding the child references
func explode(p *Package) *Package {
	var f func(*Package, *Package) *Package
	f = func(parent, p *Package) *Package {
		pkg := Package{
			Name:   p.Name,
			Parent: parent,
		}
		for _, c := range p.Children {
			pkg.Children = append(pkg.Children, f(&pkg, c))
		}
		return &pkg
	}
	return f(nil, p)
}

func pushdown(w http.ResponseWriter, r *http.Request) {
	copy := func(s []*Package) []*Package {
		c := make([]*Package, len(s))
		copy(c, s)
		return c
	}

	pkg := r.URL.Path[len("/pushdown/"):]
	root := explode(walk(pkg))
	var trim func(*Package)
	trim = func(p *Package) {
		c := copy(p.Children)
		for i := 0; i < len(c); i++ {
			fmt.Println(tree(p), "trim", c[i])
			trim(c[i])
		}
		if p.Parent == nil {
			return
		}
		for parent := p.Parent.Parent; parent != nil; parent = parent.Parent {
			fmt.Println(p, "parent", parent)
			for i := 0; i < len(parent.Children); {
				if parent.Children[i].Name == p.Name {
					fmt.Println(tree(p), "found", p, "via", tree(parent))
					parent.Children = append(parent.Children[:i], parent.Children[i+1:]...)
				} else {
					i++
				}
			}
		}
	}
	trim(root)
	enc := json.NewEncoder(w)
	enc.Encode(children(root))
}
