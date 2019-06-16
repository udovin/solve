package api

import (
	"encoding/json"
	"net/http"

	"../core"
)

type View struct {
	app *core.App
}

func Register(app *core.App, server *core.Server) {
	v := View{app: app}
	server.Handle("/api/v0/ping", v.Ping)
	server.Handle("/api/v0/user", v.CreateUser)
	server.Handle("/api/v0/user/", v.UpdateUser)
	server.Handle("/api/v0/session", v.CreateSession)
	server.Handle("/api/v0/session/", v.UpdateSession)
	server.Handle("/api/v0/problem", v.CreateProblem)
	server.Handle("/api/v0/problem/", v.UpdateProblem)
}

func (v *View) Ping(w http.ResponseWriter, r *http.Request) {
	JSONResult(w, http.StatusOK, "pong")
}

func (v *View) CreateUser(w http.ResponseWriter, r *http.Request) {
	JSONResultNotImplemented(w, r)
}

func (v *View) UpdateUser(w http.ResponseWriter, r *http.Request) {
	JSONResultNotImplemented(w, r)
}

func (v *View) DeleteUser(w http.ResponseWriter, r *http.Request) {
	JSONResultNotImplemented(w, r)
}

func (v *View) CreateSession(w http.ResponseWriter, r *http.Request) {
	JSONResultNotImplemented(w, r)
}

func (v *View) UpdateSession(w http.ResponseWriter, r *http.Request) {
	JSONResultNotImplemented(w, r)
}

func (v *View) DeleteSession(w http.ResponseWriter, r *http.Request) {
	JSONResultNotImplemented(w, r)
}

func (v *View) CreateProblem(w http.ResponseWriter, r *http.Request) {
	JSONResultNotImplemented(w, r)
}

func (v *View) UpdateProblem(w http.ResponseWriter, r *http.Request) {
	JSONResultNotImplemented(w, r)
}

func (v *View) DeleteProblem(w http.ResponseWriter, r *http.Request) {
	JSONResultNotImplemented(w, r)
}

func JSONResultNotImplemented(w http.ResponseWriter, r *http.Request) {
	JSONResult(w, http.StatusNotImplemented, map[string]string{
		"Message": "Handler not implemented yet",
	})
}

func JSONResult(w http.ResponseWriter, status int, content interface{}) {
	bytes, err := json.Marshal(content)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(bytes)
}
