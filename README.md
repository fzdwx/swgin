# Swgin

Help gin users generate swagger when defining routes

Install

```shell
go get github.com/fzdwx/swgin
```

Usage

```go
type Req struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
	List []Item `json:"list"`
}

type Item struct {
	Java   string `json:"java"`
	Golang string `json:"golang"`
}

func main() {

	e := gin.New()
	s := swgin.New(e)

	s.Router(swgin.Router{
		Method:  "Get",
		Path:    "hello",
		Summary: "测试 hello",
		Body:    Req{},
		Handlers: []gin.HandlerFunc{
			func(context *gin.Context) {

			},
		},
	})

	o := s.Parse()

	bytes, err := json.Marshal(o)
	if err != nil {
		return
	}

	err = os.WriteFile("swagger.json", bytes, 0644)
	if err != nil {
		return
	}
}
```

