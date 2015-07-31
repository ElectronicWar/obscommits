package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

func initGithub(hookpath string) {
	http.HandleFunc(hookpath, func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-Github-Event") {
		case "push":
			pushHandler(r)
		case "gollum":
			wikiHandler(r)
		case "pull_request":
			prHandler(r)
		case "issues":
			issueHandler(r)
		}
	})
}

func handlePayload(r *http.Request, data interface{}) error {
	if r.Header.Get("Content-Type") == "application/json" {
		dec := json.NewDecoder(r.Body)
		return dec.Decode(&data)
	} else {
		payload := r.FormValue("payload")
		return json.Unmarshal([]byte(payload), &data)
	}
}

func pushHandler(r *http.Request) {
	var data struct {
		Ref     string
		Commits []struct {
			Author struct {
				Username string
			}
			Url     string
			Message string
			Id      string
		}
		Repository struct {
			Name string
			Url  string
		}
	}

	err := handlePayload(r, &data)
	if err != nil {
		P("Error unmarshaling json:", err)
		return
	}

	pos := strings.LastIndex(data.Ref, "/") + 1
	branch := data.Ref[pos:]
	lines := make([]string, 0, len(data.Commits))
	repo := data.Repository.Name
	repourl := data.Repository.Url
	b := bytes.NewBuffer(nil)

	if branch != "master" {
		return
	}

	for _, v := range data.Commits {
		firstline := strings.TrimSpace(v.Message)
		pos = strings.Index(firstline, "\n")
		if pos > 0 {
			firstline = strings.TrimSpace(firstline[:pos])
		}

		b.Reset()
		tmpl.execute(b, "push", &struct {
			Author  string // commits[0].author.username
			Url     string // commits[0].url
			Message string // commits[0].message
			ID      string // commits[0].id
			Repo    string // repository.name
			Repourl string // repository.url
			Branch  string // .ref the part after refs/heads/
		}{
			Author:  v.Author.Username,
			Url:     v.Url,
			Message: firstline,
			ID:      v.Id,
			Repo:    repo,
			Repourl: repourl,
			Branch:  branch,
		})
		lines = append(lines, b.String())
	}

	if l := len(lines); l > 5 {
		lines = lines[l-5:]
	}

	go srv.handleLines("#obs-dev", lines, true)
}

func prHandler(r *http.Request) {
	var data struct {
		Action       string
		Pull_request struct {
			Html_url string
			Title    string
			User     struct {
				Login string
			}
		}
	}

	err := handlePayload(r, &data)
	if err != nil {
		P("Error unmarshaling json:", err)
		return
	}

	if data.Action != "opened" {
		return
	}

	b := bytes.NewBuffer(nil)
	tmpl.execute(b, "pr", &struct {
		Author string
		Title  string
		Url    string
	}{
		Author: data.Pull_request.User.Login,
		Title:  data.Pull_request.Title,
		Url:    data.Pull_request.Html_url,
	})

	go srv.handleLines("#obs-dev", []string{b.String()}, true)
}

func wikiHandler(r *http.Request) {
	var data struct {
		Pages []struct {
			Page_name string
			Action    string
			Sha       string
			Html_url  string
		}
		Sender struct {
			Login string
		}
	}

	err := handlePayload(r, &data)
	if err != nil {
		P("Error unmarshaling json:", err)
		return
	}

	lines := make([]string, 0, len(data.Pages))
	b := bytes.NewBuffer(nil)
	for _, v := range data.Pages {

		b.Reset()
		tmpl.execute(b, "wiki", &struct {
			Author string
			Page   string
			Url    string
			Action string
			Sha    string
		}{
			Author: data.Sender.Login,
			Page:   v.Page_name,
			Url:    v.Html_url,
			Action: v.Action,
			Sha:    v.Sha,
		})
		lines = append(lines, b.String())
	}

	if l := len(lines); l > 5 {
		lines = lines[l-5:]
	}

	go srv.handleLines("#obs-dev", lines, true)
}

func issueHandler(r *http.Request) {
	var data struct {
		Action string
		Issue  struct {
			Title    string
			Html_url string
			User     struct {
				Login string
			}
		}
	}

	err := handlePayload(r, &data)
	if err != nil {
		P("Error unmarshaling json:", err)
		return
	}

	if data.Action != "opened" {
		return
	}

	b := bytes.NewBuffer(nil)
	tmpl.execute(b, "issues", &struct {
		Author string
		Title  string
		Url    string
	}{
		Author: data.Issue.User.Login,
		Title:  data.Issue.Title,
		Url:    data.Issue.Html_url,
	})

	go srv.handleLines("#obs-dev", []string{b.String()}, true)
}
