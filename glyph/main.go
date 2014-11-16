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
	http.HandleFunc("/imports/", imports)
	http.HandleFunc("/csv/", csv)

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

	type Node struct {
		Name     string  `json:"name"`
		Children []*Node `json:"children,omitempty"`
	}
	var f func(string) *Node
	f = func(p string) *Node {
		switch p {
		case "C", "unsafe":
			return nil
		default:
			pkg, err := build.Import(p, "", 0)
			if err != nil {
				log.Fatal(err)
			}
			var ch []*Node
			for _, pkg := range pkg.Imports {
				if n := f(pkg); n != nil {
					ch = append(ch, n)
				}
			}
			return &Node{p, ch}
		}
	}

	enc := json.NewEncoder(w)
	enc.Encode(f(pkg))
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

func csv(w http.ResponseWriter, r *http.Request) {
	pkg := r.URL.Path[len("/csv/"):]
	root := Build(pkg)
	fmt.Fprintln(w, "source,target,weight")
	weight := func(p string) float64 {
		if p == pkg {
			return 2
		}
		return 1
	}
	for k, v := range packages(root) {
		for _, p := range v.Children {
			fmt.Fprintf(w, "%s,%s,%v\n", k, p.Name, weight(k))
		}
	}
}

func packages(root *Package) map[string]*Package {
	pkgs := make(map[string]*Package)
	var f func(*Package)
	f = func(p *Package) {
		pkgs[p.Name] = p
		for _, ch := range p.Children {
			f(ch)
		}
	}
	f(root)
	return pkgs
}

type Package struct {
	Name     string     `json:"name"`
	Parent   *Package   `json:"parent,omitempty"`
	Children []*Package `json:"children,omitempty"`
}

func Build(p string) *Package {
	pkgs := make(map[string]*Package)
	var f func(*Package, string) *Package
	f = func(parent *Package, p string) *Package {
		switch p {
		case "C", "unsafe":
			// skip
			return nil
		default:
			if pkg, ok := pkgs[p]; ok {
				return pkg
			}
			pkg, err := build.Import(p, "", 0)
			if err != nil {
				log.Fatal(err)
			}
			var ch []*Package
			pk := Package{
				Name:   p,
				Parent: parent,
			}
			for _, i := range pkg.Imports {
				pkg := f(&pk, i)
				if pkg != nil {
					ch = append(ch, pkg)
				}
			}
			pk.Children = ch
			return &pk
		}
	}
	return f(nil, p)
}
