package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Coaster struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Manufacturer string `json:"manufacturer"`
	InPark       string `json:"inPark"`
	Height       int    `json:"height"`
}

func newHandlers() *handlers {
	return &handlers{
		coasters: map[int]Coaster{},
	}
}

type handlers struct {
	coasters map[int]Coaster
	sync.Mutex
}

func (h *handlers) getCoastersHandler(w http.ResponseWriter, r *http.Request) {
	h.Lock()
	coasterList := make([]Coaster, len(h.coasters))
	i := 0
	for _, item := range h.coasters {
		coasterList[i] = item
		i++
	}
	h.Unlock()

	w.Header().Add("content-type", "application/json")
	asJson, err := json.MarshalIndent(coasterList, "", "  ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	w.WriteHeader(http.StatusOK)
	w.Write(asJson)
}

func (h *handlers) getCoasterHandler(w http.ResponseWriter, r *http.Request) {
	pathSegments := strings.Split(r.URL.Path, "/")
	if len(pathSegments) < 3 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	h.Lock()
	defer h.Unlock()
	id, err := strconv.Atoi(pathSegments[2])
	coaster, ok := h.coasters[id]
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Add("content-type", "application/json")
	asJson, err := json.MarshalIndent(coaster, "", "  ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(asJson)
}

func (h *handlers) randomCoaster(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	h.Lock()
	coasterList := make([]Coaster, len(h.coasters))
	i := 0
	for _, item := range h.coasters {
		coasterList[i] = item
		i++
	}
	h.Unlock()

	var coaster Coaster
	if len(coasterList) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if len(coasterList) == 1 {
		coaster = coasterList[0]
	} else {
		rand.Seed(time.Now().UnixNano())
		coaster = coasterList[rand.Intn(len(coasterList)-1)]
	}

	w.Header().Add("location", fmt.Sprintf("/coasters/%d", coaster.ID))
	w.WriteHeader(http.StatusFound)
}

func (h *handlers) postCoastersHandler(w http.ResponseWriter, r *http.Request) {
	ct := r.Header.Get("content-type")
	if ct != "application/json" {
		w.Header().Add("Accept", "application/json")
		w.WriteHeader(http.StatusUnsupportedMediaType)
		w.Write([]byte(fmt.Sprintf("status 415, unsupported content type, want 'application/json', got '%s'", ct)))
		return
	}

	asJson, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	var coaster Coaster
	err = json.Unmarshal(asJson, &coaster)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	coaster.ID = time.Now().Nanosecond()
	h.Lock()
	h.coasters[coaster.ID] = coaster
	h.Unlock()
}

func (h *handlers) coastersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.getCoastersHandler(w, r)
		return
	case "POST":
		h.postCoastersHandler(w, r)
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

type adminPortal struct {
	password string
}

func (a *adminPortal) adminHandler(w http.ResponseWriter, r *http.Request) {
	user, pass, ok := r.BasicAuth()
	if !ok || user != "admin" || pass != a.password {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(fmt.Sprintf("status 401 - unauthorized")))
		return
	}

	w.Header().Add("content-type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprint("<html><h1>Super Secret Admin Portal</h1></html>")))
}

func newAdminPortal() *adminPortal {
	pwd := os.Getenv("ADMIN_PASSWORD")
	if pwd == "" {
		fmt.Println("no admin password (env var ADMIN_PASSWORD) set")
		os.Exit(1)
	}

	return &adminPortal{
		password: pwd,
	}
}

func main() {
	handlers := newHandlers()
	admin := newAdminPortal()
	http.HandleFunc("/admin", admin.adminHandler)
	http.HandleFunc("/coasters", handlers.coastersHandler)
	http.HandleFunc("/coasters/", handlers.getCoasterHandler)
	http.HandleFunc("/coasters/random", handlers.randomCoaster)
	http.ListenAndServe(":8080", nil)
}
