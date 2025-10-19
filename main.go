package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Participant struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
	Photo string `json:"photo,omitempty"`
}

// фиксированные участники
var participants = map[string]*Participant{
	"vova":   {ID: "vova", Name: "Вова", Count: 0, Photo: "/photos/vova.jpg"},
	"misha":  {ID: "misha", Name: "Миша", Count: 0, Photo: "/photos/misha.jpg"},
	"stepa":  {ID: "stepa", Name: "Стёпа", Count: 0, Photo: "/photos/stepa.jpg"},
	"egor":   {ID: "egor", Name: "Егор", Count: 0, Photo: "/photos/egor.jpg"},
	"timur":  {ID: "timur", Name: "Тимур", Count: 0, Photo: "/photos/timur.jpg"},
	"kp":     {ID: "kp", Name: "КатяПолина", Count: 0, Photo: "/photos/kp.jpg"},
	"timoha": {ID: "timoha", Name: "Тимоха", Count: 0, Photo: "/photos/timoha.jpg"},
	"igor":   {ID: "igor", Name: "Игорь", Count: 0, Photo: "/photos/igor.jpg"},
	"makar":  {ID: "makar", Name: "Макар", Count: 0, Photo: "/photos/makar.jpg"},
}

var secretPass string

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// вернуть всех участников
func getParticipants(w http.ResponseWriter, r *http.Request) {
	list := make([]*Participant, 0, len(participants))
	for _, p := range participants {
		list = append(list, p)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Count > list[j].Count || list[i].Count == list[j].Count && list[i].ID > list[j].ID
	})
	writeJSON(w, list)
}

// обработка /{pass}/{id}/up или /{pass}/{id}/{count}
func participantHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		http.Error(w, "not found", 404)
		return
	}

	pass := parts[0]
	id := parts[1]
	action := parts[2]

	if pass != secretPass {
		http.Error(w, "not found", 404)
		return
	}

	p, ok := participants[id]
	if !ok {
		http.Error(w, "participant not found", 404)
		return
	}

	if action == "up" {
		p.Count++
		writeJSON(w, p)
		return
	}

	// если третий параметр — число
	if count, err := strconv.Atoi(action); err == nil {
		p.Count = count
		writeJSON(w, p)
		return
	}

	http.Error(w, "invalid endpoint", 400)
}

func main() {
	// Загружаем .env
	_ = godotenv.Load()
	secretPass = os.Getenv("PASS")
	if secretPass == "" {
		fmt.Println("⚠️  WARNING: PASS not set in .env, using default 'admin'")
		secretPass = "admin"
	}

	// фронт: статика
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)

	// API
	http.HandleFunc("/leaders", getParticipants)
	http.HandleFunc("/"+secretPass+"/", participantHandler)

	fmt.Println("Server running on :1337")
	fmt.Printf("Use URL: /%s/<id>/up or /%s/<id>/<count>\n", secretPass, secretPass)
	http.ListenAndServe(":1337", nil)
}
