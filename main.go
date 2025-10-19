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

// LeaderResponse — то, что мы отправляем клиенту для каждого участника
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
	// Защита конкурентного доступа
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

// getParticipantsHandler возвращает отсортированный по Count список
// вместе с полем contest_start
func getParticipantsHandler(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()

	// соберём срез
	list := make([]*Participant, 0, len(participants))
	for _, p := range participants {
		list = append(list, p)
	}
	// сортируем по Count (убывание), затем по name для стабильности
	sort.Slice(list, func(i, j int) bool {
		if list[i].Count == list[j].Count {
			return list[i].Name < list[j].Name
		}
		return list[i].Count > list[j].Count
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

// startHandler сбрасывает счётчики, выставляет contestStart = now и
// для каждого участника делает LastEat = now, обнуляет статистику
func startHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// handler зарегистрирован под /{pass}/start, поэтому пароль уже проверен по пути
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

// participantActionHandler ожидает путь вида: /{pass}/{id}/up  или /{pass}/{id}/{count}
// Регистрация в main будет сделана с префиксом "/"+secretPass+"/"
func participantActionHandler(w http.ResponseWriter, r *http.Request) {
	// ожидаем POST
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// получим путь после префикса /{pass}/
	// пример r.URL.Path: "/mysecret/vova/up"
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	// parts[0] == pass, parts[1] == id, parts[2] == action
	if len(parts) < 3 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	// проверка пароля не нужна здесь, так как мы будем регистрировать хендлер именно по префиксу "/"+secretPass+"/"
	// но на всякий случай проверим
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

	// действие "up"
	if action == "up" {
		// если contestStart нулевой — автоматически инициализируем его сейчас и установим LastEat = now
		if contestStart.IsZero() {
			contestStart = now
			// чтобы первый interval считался от старта, выставим всем LastEat = now
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
			// учитываем интервал в статистике
			p.TotalIntervals += interval
			p.IntervalCount++
			if p.BestTime == 0 || interval < p.BestTime {
				p.BestTime = interval
			}
		}
		p.LastEat = now
		p.Count++

		fmt.Printf("[%s] %s up -> count=%d interval_ms=%d\n", now.Format(time.RFC3339), p.ID, p.Count, int64(interval/time.Millisecond))

		// вернуть обновлённого участника в виде JSON-ответа
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

	// если action — число: установить count
	if n, err := strconv.Atoi(action); err == nil {
		// при ручной установке счёта — мы не трогаем временные метрики,
		// чтобы не ломать историю; если нужно сбросить времена, используйте /{pass}/start
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
	// load .env if present
	_ = godotenv.Load()
	secretPass = os.Getenv("PASS")
	if secretPass == "" {
		fmt.Println("⚠️ PASS not set in .env — using default 'admin' (consider setting PASS in .env)")
		secretPass = "admin"
	}

	// Handlers:
	// GET /leaders
	http.HandleFunc("/leaders", getParticipantsHandler)

	// POST /{pass}/start
	http.HandleFunc("/"+secretPass+"/start", startHandler)

	// POST /{pass}/{id}/up  and POST /{pass}/{id}/{count}
	// register handler on prefix "/{pass}/"
	http.HandleFunc("/"+secretPass+"/", participantActionHandler)

	// static files (frontend) - should be last so /leaders and /{pass}/... matched first when relevant
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)

	fmt.Println("Server running on :1337")
	fmt.Printf("Use endpoints:\n - GET /leaders\n - POST /%s/start\n - POST /%s/{id}/up\n - POST /%s/{id}/{count}\n", secretPass, secretPass, secretPass)
	if err := http.ListenAndServe(":1337", nil); err != nil {
		fmt.Fprintf(os.Stderr, "server failed: %v\n", err)
		os.Exit(1)
	}
}
