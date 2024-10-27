package api

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/udovin/solve/internal/managers"
)

func TestPostSimpleScenario(t *testing.T) {
	e := NewTestEnv(t)
	defer e.Close()
	user := NewTestUser(e)
	user.AddRoles("observe_posts", "create_post")
	user.LoginClient()
	{
		file, err := os.Open(filepath.Join(testDataDir, "a-plus-b.zip"))
		if err != nil {
			t.Fatal("Error:", err)
		}
		form := CreatePostForm{
			Title:       "Problem package",
			Description: "Problem package \\url{a-plus-b.zip}",
			Files: []PostFormFile{
				{
					Name:    "a-plus-b.zip",
					Content: managers.NewFileReader(file),
				},
			},
		}
		post, err := e.Client.CreatePost(context.Background(), form)
		if err != nil {
			t.Fatal("Error:", err)
		} else {
			e.Check(post)
		}
	}
}
