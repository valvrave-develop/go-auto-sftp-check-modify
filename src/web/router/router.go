package router

import (
	"errors"
	"web/resource"
)

var RouterTable Router = new(router)

type Router interface {
	Register(string, resource.Resource)
	GetRouter(string) (resource.Resource, error)
}

type router struct {
	routerTable map[string]resource.Resource
}

func (r *router) Register(project string, handle resource.Resource) {
	if r.routerTable == nil {
		r.routerTable = make(map[string]resource.Resource)
	}
	r.routerTable[project] = handle
}

func (r *router) GetRouter(project string) (resource.Resource, error) {
	 if _, ok := r.routerTable[project]; ok {
	 	return r.routerTable[project], nil
	 }
	 return nil, errors.New("Not Found")
}