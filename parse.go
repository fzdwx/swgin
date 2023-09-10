package swgin

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

func parse(sg *Swgin) swaggerObject {
	s := swaggerObject{
		Swagger:           "2.0",
		Schemes:           []string{"http", "https"},
		Consumes:          []string{"application/json"},
		Produces:          []string{"application/json"},
		Paths:             make(swaggerPathsObject),
		Definitions:       make(swaggerDefinitionsObject),
		StreamDefinitions: make(swaggerDefinitionsObject),
		Info: swaggerInfoObject{
			Title:       sg.Title,
			Version:     sg.Version,
			Description: sg.Description,
		},
	}

	if len(sg.Host) > 0 {
		s.Host = sg.Host
	}
	if len(sg.BasePath) > 0 {
		s.BasePath = sg.BasePath
	}

	requestResponseRefs := refMap{}
	renderRouters(s.Paths, s.Definitions, requestResponseRefs, sg)
	return s
}

func renderRouters(paths swaggerPathsObject, definitions swaggerDefinitionsObject, refs refMap, sg *Swgin) {
	for i := range sg.routers {
		router := sg.routers[i]
		renderRouter(paths, definitions, refs, router)
	}

	for i := range sg.groups {
		group := sg.groups[i]
		for j := range group.Routers {
			router := group.Routers[j]
			router.Path = group.Path + router.Path
			renderRouter(paths, definitions, refs, router)
		}
	}
}

func renderRouter(paths swaggerPathsObject, definitions swaggerDefinitionsObject, requestResponseRefs refMap, router Router) {
	renderReplyAsDefinition(definitions, router, requestResponseRefs)
	parameters := swaggerParametersObject{}
	path := router.Path

	pathPairs := strings.Split(path, "/")
	for i := range pathPairs {
		parameters = parsePathParameters(pathPairs[i], path, parameters, router)
	}

	if router.Query != nil {
		parameters = append(parameters, parseQueryOrBody(router.Query, "query"))
	}
	if router.Body != nil {
		parameters = append(parameters, parseQueryOrBody(router.Body, "body"))
	}

	pathItemObject, ok := paths[path]
	if !ok {
		pathItemObject = swaggerPathItemObject{}
	}

	desc := "A successful response."
	respSchema := schemaCore{}
	respTypeName := typeName(router.ResponseType)
	if router.ResponseType != nil && len(respTypeName) > 0 {
		if strings.HasPrefix(respTypeName, "[]") {

			refTypeName := strings.Replace(respTypeName, "[", "", 1)
			refTypeName = strings.Replace(refTypeName, "]", "", 1)

			respSchema.Type = "array"
			respSchema.Items = &swaggerItemsObject{Ref: fmt.Sprintf("#/definitions/%s", refTypeName)}
		} else {
			respSchema.Ref = fmt.Sprintf("#/definitions/%s", respTypeName)
		}
	}
	operationObject := &swaggerOperationObject{
		Tags:       router.Tags,
		Parameters: parameters,
		Responses: swaggerResponsesObject{
			"200": swaggerResponseObject{
				Description: desc,
				Schema: swaggerSchemaObject{
					schemaCore: respSchema,
				},
			},
		},
	}

	for _, param := range operationObject.Parameters {
		if param.Schema != nil && param.Schema.Ref != "" {
			requestResponseRefs[param.Schema.Ref] = struct{}{}
		}
	}
	operationObject.Summary = router.Summary
	operationObject.Description = router.Description
	switch strings.ToUpper(router.Method) {
	case http.MethodGet:
		pathItemObject.Get = operationObject
	case http.MethodPost:
		pathItemObject.Post = operationObject
	case http.MethodDelete:
		pathItemObject.Delete = operationObject
	case http.MethodPut:
		pathItemObject.Put = operationObject
	case http.MethodPatch:
		pathItemObject.Patch = operationObject
	}
	paths[path] = pathItemObject
}

func parseQueryOrBody(a any, name string) swaggerParameterObject {
	t := reflect.TypeOf(a)

	reqRef := fmt.Sprintf("#/definitions/%s", t.Name())
	schema := swaggerSchemaObject{
		schemaCore: schemaCore{
			Ref: reqRef,
		},
	}

	parameter := swaggerParameterObject{
		Name:     name,
		In:       name,
		Required: true,
		Schema:   &schema,
	}
	return parameter
}

func parsePathParameters(part, path string, parameters swaggerParametersObject, router Router) swaggerParametersObject {
	if strings.Contains(part, ":") {
		key := strings.TrimPrefix(part, ":")
		path = strings.Replace(path, fmt.Sprintf(":%s", key), fmt.Sprintf("{%s}", key), 1)

		spo := swaggerParameterObject{
			Name:     key,
			In:       "path",
			Required: true,
			Type:     "string",
		}

		prop := router.Properties[key]
		if prop != "" {
			// remove quotes
			spo.Description = strings.Trim(prop, "\"")
		}

		parameters = append(parameters, spo)
	}

	return parameters
}

func typeName(a any) string {
	if a == nil {
		return ""
	}
	return reflect.TypeOf(a).Name()
}

func renderReplyAsDefinition(d swaggerDefinitionsObject, router Router, refs refMap) {
	if router.Body != nil {
		toDefinition(d, router.Body)
	}

	if router.Query != nil {
		toDefinition(d, router.Query)
	}

	if router.ResponseType != nil {
		toDefinition(d, router.ResponseType)
	}
}

func toDefinition(d swaggerDefinitionsObject, t any) {
	schema := swaggerSchemaObject{
		schemaCore: schemaCore{
			Type: "object",
		},
	}

	var typeOf reflect.Type
	switch t.(type) {
	case reflect.Type:
		typeOf = t.(reflect.Type)
	default:
		typeOf = reflect.TypeOf(t)
	}
	name := typeOf.Name()

	schema.Title = name

	for i := 0; i < typeOf.NumField(); i++ {
		member := typeOf.Field(i)
		if member.Type.Kind() == reflect.Struct {
			toDefinition(d, member.Type)
		}

		kv := keyVal{Value: schemaOfField(member, d)}
		kv.Key = member.Name
		if schema.Properties == nil {
			schema.Properties = &swaggerSchemaObjectProperties{}
		}
		*schema.Properties = append(*schema.Properties, kv)

		structTag := member.Tag
		if structTag.Get("required") == "true" && !contains(schema.Required, member.Name) {
			schema.Required = append(schema.Required, member.Name)
		}
	}

	d[name] = schema
}

func schemaOfField(member reflect.StructField, d swaggerDefinitionsObject) swaggerSchemaObject {
	ret := swaggerSchemaObject{}

	var core schemaCore

	kind := member.Type.Kind()
	var props *swaggerSchemaObjectProperties

	comment := member.Tag.Get("comment")
	switch ft := kind; ft {
	case reflect.Invalid: //[]Struct 也有可能是 Struct
		// []Struct
		// map[ArrayType:map[Star:map[StringExpr:UserSearchReq] StringExpr:*UserSearchReq] StringExpr:[]*UserSearchReq]
		refTypeName := strings.Replace(member.Type.Name(), "[", "", 1)
		refTypeName = strings.Replace(refTypeName, "]", "", 1)
		refTypeName = strings.Replace(refTypeName, "*", "", 1)
		refTypeName = strings.Replace(refTypeName, "{", "", 1)
		refTypeName = strings.Replace(refTypeName, "}", "", 1)
		// interface

		if refTypeName == "interface" {
			core = schemaCore{Type: "object"}
		} else if refTypeName == "mapstringstring" {
			core = schemaCore{Type: "object"}
		} else if strings.HasPrefix(refTypeName, "[]") {
			core = schemaCore{Type: "array"}

			tempKind := swaggerMapTypes[strings.Replace(refTypeName, "[]", "", -1)]
			ftype, format, ok := primitiveSchema(tempKind, refTypeName)
			if ok {
				core.Items = &swaggerItemsObject{Type: ftype, Format: format}
			} else {
				core.Items = &swaggerItemsObject{Type: ft.String(), Format: "UNKNOWN"}
			}

		} else {
			core = schemaCore{
				Ref: "#/definitions/" + refTypeName,
			}
		}
	case reflect.Slice:
		elem := member.Type.Elem()
		name := elem.Name()
		tempKind := swaggerMapTypes[strings.Replace(member.Type.Name(), "[]", "", -1)]
		if tempKind == reflect.Invalid {
			core = schemaCore{Ref: "#/definitions/" + name, Format: name}
			toDefinition(d, member.Type.Elem())
		} else {
			ftype, format, ok := primitiveSchema(tempKind, member.Type.Name())

			if ok {
				core = schemaCore{Type: ftype, Format: format}
			} else {
				core = schemaCore{Type: ft.String(), Format: "UNKNOWN"}
			}
		}
	default:
		ftype, format, ok := primitiveSchema(ft, member.Type.Name())
		if ok {
			core = schemaCore{Type: ftype, Format: format}
		} else {
			core = schemaCore{Type: ft.String(), Format: "UNKNOWN"}
		}
	}

	switch ft := kind; ft {
	case reflect.Slice:
		ret = swaggerSchemaObject{
			schemaCore: schemaCore{
				Type:  "array",
				Items: (*swaggerItemsObject)(&core),
			},
		}
	case reflect.Invalid:
		// 判断是否数组
		if strings.HasPrefix(member.Type.Name(), "[]") {
			ret = swaggerSchemaObject{
				schemaCore: schemaCore{
					Type:  "array",
					Items: (*swaggerItemsObject)(&core),
				},
			}
		} else {
			ret = swaggerSchemaObject{
				schemaCore: core,
				Properties: props,
			}
		}
		if strings.HasPrefix(member.Type.Name(), "map") {
			fmt.Println("暂不支持map类型")
		}
	default:
		ret = swaggerSchemaObject{
			schemaCore: core,
			Properties: props,
		}
	}
	ret.Description = comment

	return ret
}

// https://swagger.io/specification/ Data Types
func primitiveSchema(kind reflect.Kind, t string) (ftype, format string, ok bool) {
	switch kind {
	case reflect.Int:
		return "integer", "int32", true
	case reflect.Uint:
		return "integer", "uint32", true
	case reflect.Int8:
		return "integer", "int8", true
	case reflect.Uint8:
		return "integer", "uint8", true
	case reflect.Int16:
		return "integer", "int16", true
	case reflect.Uint16:
		return "integer", "uin16", true
	case reflect.Int64:
		return "integer", "int64", true
	case reflect.Uint64:
		return "integer", "uint64", true
	case reflect.Bool:
		return "boolean", "boolean", true
	case reflect.String:
		return "string", "", true
	case reflect.Float32:
		return "number", "float", true
	case reflect.Float64:
		return "number", "double", true
	case reflect.Slice:
		return strings.Replace(t, "[]", "", -1), "", true
	default:
		return "", "", false
	}
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}
