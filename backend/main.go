package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Participant struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

var participants = map[string]*Participant{
	"1": {ID: "1", Name: "Иван", Count: 0},
	"2": {ID: "2", Name: "Оля", Count: 0},
}

// отдать всех участников
func getLeaders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	list := []*Participant{}
	for _, p := range participants {
		list = append(list, p)
	}
	json.NewEncoder(w).Encode(list)
}

// увеличить счётчик
func upHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/up/"):]
	if p, ok := participants[id]; ok {
		p.Count++
		fmt.Fprintf(w, "ok")
	} else {
		http.Error(w, "not found", 404)
	}
}

func main() {
	// статические файлы (HTML, JS, CSS)
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)

	// API
	http.HandleFunc("/leaders", getLeaders)
	http.HandleFunc("/up/", upHandler)

	fmt.Println("Server running on :8080")
	http.ListenAndServe(":8080", nil)
}
