package api

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/udovin/solve/api/schema"
	"github.com/udovin/solve/internal/managers"
)

func TestPostSimpleScenario(t *testing.T) {
	e := NewTestEnv(t)
	defer e.Close()
	owner := NewTestUser(e)
	owner.AddRoles("observe_posts", "create_post")
	post := func() schema.Post {
		owner.LoginClient()
		defer owner.LogoutClient()
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
		resp, err := e.Client.CreatePost(context.Background(), form)
		if err != nil {
			t.Fatal("Error:", err)
		} else {
			e.Check(resp)
		}
		return resp
	}()
	if _, err := e.Client.ObservePost(context.Background(), ObservePostRequest{ID: post.ID}); err == nil {
		t.Fatal("Expected error")
	} else {
		e.Check(err)
	}
	func() {
		owner.LoginClient()
		defer owner.LogoutClient()
		form := UpdatePostForm{
			Publish: getPtr(true),
		}
		resp, err := e.Client.UpdatePost(context.Background(), post.ID, form)
		if err != nil {
			t.Fatal("Error:", err)
		} else {
			e.Check(resp)
		}
	}()
	if resp, err := e.Client.ObservePost(context.Background(), ObservePostRequest{ID: post.ID}); err != nil {
		t.Fatal("Error:", err)
	} else {
		e.Check(resp)
	}
	user := NewTestUser(e)
	user.AddRoles("observe_posts")
	func() {
		user.LoginClient()
		defer user.LogoutClient()
		if resp, err := e.Client.ObservePost(context.Background(), ObservePostRequest{ID: post.ID}); err != nil {
			t.Fatal("Error:", err)
		} else {
			e.Check(resp)
		}
	}()
	func() {
		owner.LoginClient()
		defer owner.LogoutClient()
		if resp, err := e.Client.ObservePost(context.Background(), ObservePostRequest{
			ID:        post.ID,
			WithFiles: true,
		}); err != nil {
			t.Fatal("Error:", err)
		} else {
			e.Check(resp)
		}
	}()
}
