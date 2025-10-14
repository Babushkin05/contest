package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

type Participant struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
	Photo string `json:"photo,omitempty"`
}

// фиксированные участники
var participants = map[string]*Participant{
	"vova":  {ID: "vova", Name: "Вова", Count: 0, Photo: "/photos/vova.jpg"},
	"misha": {ID: "misha", Name: "Миша", Count: 0, Photo: "/photos/misha.jpg"},
	"stepa": {ID: "stepa", Name: "Степа", Count: 0, Photo: "/photos/stepa.jpg"},
	"egor":  {ID: "egor", Name: "Егор", Count: 0, Photo: "/photos/egor.jpg"},
	"timur": {ID: "timur", Name: "Тимур", Count: 0, Photo: "/photos/timur.jpg"},
}

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
		return list[i].ID < list[j].ID
	})
	writeJSON(w, list)
}

// обработка /{id}/up или /{id}/{count}
func participantHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) == 0 {
		http.Error(w, "not found", 404)
		return
	}
	id := parts[0]
	p, ok := participants[id]
	if !ok {
		http.Error(w, "participant not found", 404)
		return
	}

	if len(parts) == 2 {
		// /vova/up или /vova/5
		if parts[1] == "up" {
			p.Count++
			writeJSON(w, p)
			return
		}
		// попытка установить конкретное количество
		if count, err := strconv.Atoi(parts[1]); err == nil {
			p.Count = count
			writeJSON(w, p)
			return
		}
	}

	http.Error(w, "invalid endpoint", 400)
}

func main() {
	// фронт: статические файлы
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)

	// API: супер простой
	for id := range participants {
		http.HandleFunc("/"+id+"/", participantHandler)
	}

	// вернуть всех участников
	http.HandleFunc("/leaders", getParticipants)

	fmt.Println("Server running on http://localhost:1337")
	http.ListenAndServe(":1337", nil)
}
