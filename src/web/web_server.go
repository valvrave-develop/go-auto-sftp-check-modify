package web

import (
	"fmt"
	"net/http"
	"project"
	"strings"
	"web/resource"
	"web/router"
)

type HttpServerHandle struct {
	projects map[string]*project.Project
}

func (h *HttpServerHandle) RegisterRouters() {
	for key, value := range h.projects {
		router.RouterTable.Register(key, resource.NewDirTreeResource(value))
	}
}

func (h *HttpServerHandle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if len(r.URL.Path) < 2 {
		w.Write([]byte("Invalid url"))
		return
	}
	handle, err := router.RouterTable.GetRouter(r.URL.Path[1:])
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	keys := strings.Split(r.URL.RawQuery, "=")
	if len(keys) != 2 || keys[0] != "path" {
		w.Write([]byte("Invalid url"))
		return
	}
	w.Write([]byte(handle.Get(keys[1])))
}

func WebServerStart(projects map[string]*project.Project){
	handle := &HttpServerHandle{projects:projects}
	handle.RegisterRouters()
	if err := http.ListenAndServe(":7090", handle); err != nil {
		fmt.Println("http server listen failed:", err)
	}

}
