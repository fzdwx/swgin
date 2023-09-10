package swgin

import (
	"github.com/gin-gonic/gin"
	"strings"
)

type Swgin struct {
	e gin.IRouter

	Title       string
	Version     string
	Description string
	Host        string
	BasePath    string

	groups  []RouterGroup
	routers []Router
}

func New(e *gin.Engine) *Swgin {
	return &Swgin{
		e:       e,
		groups:  []RouterGroup{},
		routers: []Router{},
	}
}

func (s *Swgin) Parse() swaggerObject {
	return parse(s)
}

func (s *Swgin) Group(rg RouterGroup) *Swgin {
	group := s.e.Group(rg.Path)
	for i := range rg.Routers {
		router := rg.Routers[i]
		s.router(router.Method, router.Path, group, router.Handlers...)
	}

	s.groups = append(s.groups, rg)

	return s
}

func (s *Swgin) Router(r Router) *Swgin {
	s.routers = append(s.routers, r)

	return s.router(r.Method, r.Path, s.e, r.Handlers...)
}

func (s *Swgin) router(method, path string, r gin.IRouter, handlers ...gin.HandlerFunc) *Swgin {
	method = strings.ToUpper(method)

	switch method {
	case "GET":
		r.GET(path, handlers...)
	case "POST":
		r.POST(path, handlers...)
	case "PUT":
		r.PUT(path, handlers...)
	case "PATCH":
		r.PATCH(path, handlers...)
	case "DELETE":
		r.DELETE(path, handlers...)
	case "OPTIONS":
		r.OPTIONS(path, handlers...)
	case "HEAD":
		r.HEAD(path, handlers...)
	default:
		r.GET(path, handlers...)
	}

	return s
}

type RouterGroup struct {
	Path    string
	Routers []Router
}

type Router struct {
	Path        string
	Method      string
	Summary     string
	Description string

	Tags         []string
	Query        any
	Body         any
	ResponseType any

	Handlers []gin.HandlerFunc

	// extended properties
	Properties map[string]string
}
