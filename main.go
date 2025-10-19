package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

type Participant struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	Count          int           `json:"count"`
	Photo          string        `json:"photo,omitempty"`
	LastEat        time.Time     `json:"-"`
	BestTime       time.Duration `json:"-"`
	TotalIntervals time.Duration `json:"-"`
	IntervalCount  int           `json:"-"`
}

type LeaderResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Count         int    `json:"count"`
	Photo         string `json:"photo,omitempty"`
	BestMs        int64  `json:"best_ms"`        // миллисекунды, 0 если нет
	AvgMs         int64  `json:"avg_ms"`         // миллисекунды, 0 если нет
	IntervalCount int    `json:"interval_count"` // число интервалов
	LastEat       string `json:"last_eat"`       // ISO8601 или empty
}

type LeadersPayload struct {
	ContestStart string           `json:"contest_start"` // ISO8601 или empty
	Leaders      []LeaderResponse `json:"leaders"`
}

var (
	participants = map[string]*Participant{
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
	mu           sync.RWMutex
	contestStart time.Time

	secretPass string
)

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func getParticipantsHandler(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()

	list := make([]*Participant, 0, len(participants))
	for _, p := range participants {
		list = append(list, p)
	}

	// сортировка: Count (убывание) → LastEat (убывание) → Name
	sort.Slice(list, func(i, j int) bool {
		if list[i].Count != list[j].Count {
			return list[i].Count > list[j].Count
		}
		if !list[i].LastEat.Equal(list[j].LastEat) {
			return list[j].LastEat.After(list[i].LastEat)
		}
		return list[i].Name < list[j].Name
	})

	resp := LeadersPayload{
		ContestStart: "",
		Leaders:      make([]LeaderResponse, 0, len(list)),
	}
	if !contestStart.IsZero() {
		resp.ContestStart = contestStart.UTC().Format(time.RFC3339Nano)
	}

	for _, p := range list {
		var bestMs int64
		var avgMs int64
		if p.BestTime > 0 {
			bestMs = int64(p.BestTime / time.Millisecond)
		}
		if p.IntervalCount > 0 {
			avgMs = int64(p.TotalIntervals / time.Duration(p.IntervalCount) / time.Millisecond)
		}
		lastEat := ""
		if !p.LastEat.IsZero() {
			lastEat = p.LastEat.UTC().Format(time.RFC3339Nano)
		}
		resp.Leaders = append(resp.Leaders, LeaderResponse{
			ID:            p.ID,
			Name:          p.Name,
			Count:         p.Count,
			Photo:         p.Photo,
			BestMs:        bestMs,
			AvgMs:         avgMs,
			IntervalCount: p.IntervalCount,
			LastEat:       lastEat,
		})
	}

	writeJSON(w, resp)
}

func startHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	now := time.Now().UTC()
	mu.Lock()
	contestStart = now
	for _, p := range participants {
		p.Count = 0
		p.LastEat = now
		p.BestTime = 0
		p.TotalIntervals = 0
		p.IntervalCount = 0
	}
	mu.Unlock()

	fmt.Printf("[%s] contest started (reset counts and times)\n", now.Format(time.RFC3339))
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "ok")
}

func participantActionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if parts[0] != secretPass {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	id := parts[1]
	action := parts[2]

	mu.Lock()
	defer mu.Unlock()

	p, ok := participants[id]
	if !ok {
		http.Error(w, "participant not found", http.StatusNotFound)
		return
	}

	now := time.Now().UTC()

	if action == "up" {
		if contestStart.IsZero() {
			contestStart = now
			for _, q := range participants {
				q.LastEat = now
				q.BestTime = 0
				q.TotalIntervals = 0
				q.IntervalCount = 0
				q.Count = 0
			}
			fmt.Printf("[%s] contestStart auto-initialized by first up\n", now.Format(time.RFC3339))
		}

		var interval time.Duration
		if !p.LastEat.IsZero() {
			interval = now.Sub(p.LastEat)
			p.TotalIntervals += interval
			p.IntervalCount++
			if p.BestTime == 0 || interval < p.BestTime {
				p.BestTime = interval
			}
		}
		p.LastEat = now
		p.Count++

		fmt.Printf("[%s] %s up -> count=%d interval_ms=%d\n", now.Format(time.RFC3339), p.ID, p.Count, int64(interval/time.Millisecond))

		var bestMs int64
		var avgMs int64
		if p.BestTime > 0 {
			bestMs = int64(p.BestTime / time.Millisecond)
		}
		if p.IntervalCount > 0 {
			avgMs = int64(p.TotalIntervals / time.Duration(p.IntervalCount) / time.Millisecond)
		}
		lastEat := ""
		if !p.LastEat.IsZero() {
			lastEat = p.LastEat.UTC().Format(time.RFC3339Nano)
		}

		writeJSON(w, LeaderResponse{
			ID:            p.ID,
			Name:          p.Name,
			Count:         p.Count,
			Photo:         p.Photo,
			BestMs:        bestMs,
			AvgMs:         avgMs,
			IntervalCount: p.IntervalCount,
			LastEat:       lastEat,
		})
		return
	}

	if n, err := strconv.Atoi(action); err == nil {
		p.Count = n
		fmt.Printf("[%s] %s set count=%d (manual)\n", now.Format(time.RFC3339), p.ID, p.Count)

		var bestMs int64
		var avgMs int64
		if p.BestTime > 0 {
			bestMs = int64(p.BestTime / time.Millisecond)
		}
		if p.IntervalCount > 0 {
			avgMs = int64(p.TotalIntervals / time.Duration(p.IntervalCount) / time.Millisecond)
		}
		lastEat := ""
		if !p.LastEat.IsZero() {
			lastEat = p.LastEat.UTC().Format(time.RFC3339Nano)
		}

		writeJSON(w, LeaderResponse{
			ID:            p.ID,
			Name:          p.Name,
			Count:         p.Count,
			Photo:         p.Photo,
			BestMs:        bestMs,
			AvgMs:         avgMs,
			IntervalCount: p.IntervalCount,
			LastEat:       lastEat,
		})
		return
	}

	http.Error(w, "invalid endpoint", http.StatusBadRequest)
}

func main() {
	_ = godotenv.Load()
	secretPass = os.Getenv("PASS")
	if secretPass == "" {
		fmt.Println("⚠️ PASS not set in .env — using default 'admin'")
		secretPass = "admin"
	}

	http.HandleFunc("/leaders", getParticipantsHandler)
	http.HandleFunc("/"+secretPass+"/start", startHandler)
	http.HandleFunc("/"+secretPass+"/", participantActionHandler)

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)

	fmt.Println("Server running on :1337")
	fmt.Printf("Use endpoints:\n - GET /leaders\n - POST /%s/start\n - POST /%s/{id}/up\n - POST /%s/{id}/{count}\n", secretPass, secretPass, secretPass)
	if err := http.ListenAndServe(":1337", nil); err != nil {
		fmt.Fprintf(os.Stderr, "server failed: %v\n", err)
		os.Exit(1)
	}
}
