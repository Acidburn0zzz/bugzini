package main

import (
	"bugzilla"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
	"time"
)

func BugGet(id int) (*bugzilla.Bug, error) {
	if bug, ok := cache.bugsMap[id]; ok {
		return bug, nil
	}

	client, err := Bz()

	if err != nil {
		return nil, err
	}

	bug, err := client.Bugs().Get(client, id)

	if err != nil {
		return nil, err
	}

	cache.bugsMap[bug.Id] = &bug
	cache.Save()

	return &bug, nil
}

func BugHandler(w http.ResponseWriter, r *http.Request) {
	noCache(w)

	vars := mux.Vars(r)

	idl, err := strconv.ParseInt(vars["id"], 10, 32)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := int(idl)

	bug, err := BugGet(id)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	JsonResponse(w, *bug)
}

func BugCommentHandler(w http.ResponseWriter, r *http.Request) {
	noCache(w)

	if bugzilla.CurrentUser() == nil {
		http.Error(w, "Not logged in", http.StatusBadRequest)
		return
	}

	comment := r.FormValue("comment")

	if len(comment) == 0 {
		http.Error(w, "No comment specified", http.StatusBadRequest)
		return
	}

	vars := mux.Vars(r)

	idl, err := strconv.ParseInt(vars["id"], 10, 32)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := int(idl)

	client, err := Bz()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err := client.Bugs().AddComment(client, id, comment); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func BugCommentsHandler(w http.ResponseWriter, r *http.Request) {
	noCache(w)

	vars := mux.Vars(r)

	idl, err := strconv.ParseInt(vars["id"], 10, 32)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := int(idl)

	bug, err := BugGet(id)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	after := r.FormValue("after")

	// Only use cache if not asking for bugs after a certain date
	if len(after) == 0 {
		if len(bug.Comments) != 0 {
			JsonResponse(w, bug.Comments)
			return
		}
	}

	client, err := Bz()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var comments []bugzilla.Comment

	if len(after) == 0 {
		comments, err = client.Bugs().GetComments(client, id)
	} else {
		afsec, err := strconv.ParseInt(after, 10, 64)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		t := time.Unix(afsec, 0)
		comments, err = client.Bugs().GetCommentsAfter(client, id, t)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, v := range comments {
		if v.Time.After(bug.LastChangeTime) {
			bug.LastChangeTime = v.Time
		}
	}

	if len(after) == 0 {
		bug.Comments = comments
	} else if len(bug.Comments) > 0 {
		last := bug.Comments[len(bug.Comments)-1]

		for _, v := range comments {
			if v.Time.After(last.Time) {
				bug.Comments = append(bug.Comments, v)
			}
		}
	}

	cache.Save()
	JsonResponse(w, comments)
}

func init() {
	router.HandleFunc("/api/bug/{id:[0-9]+}/comments", BugCommentsHandler)
	router.HandleFunc("/api/bug/{id:[0-9]+}/comment", BugCommentHandler)
	router.HandleFunc("/api/bug/{id:[0-9]+}", BugHandler)
}
