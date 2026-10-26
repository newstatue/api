package main

import (
	"context"
	"log"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

// GetUserInput GET /users/{id} 请求参数
type GetUserInput struct {
	ID int `path:"id"`
}

// GetUserOutput GET /users/{id} 响应
type GetUserOutput struct {
	Body User
}

// User 响应数据模型
type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func main() {
	mux := http.NewServeMux()
	api := humago.New(mux, huma.DefaultConfig("示例API", "0.0.1"))
	huma.Get(api, "/users/{id}", func(ctx context.Context, i *GetUserInput) (*GetUserOutput, error) {
		return &GetUserOutput{Body: User{
			ID:   i.ID,
			Name: "hello",
		}}, nil
	})

	addr := ":8080"
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	log.Printf("服务器运行于: %s", addr)
	log.Fatal(server.ListenAndServe())
}
