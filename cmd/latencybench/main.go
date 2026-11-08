// Command latencybench measures realtime delivery latency end to end: an
// outbox event is committed to PostgreSQL, picked up by the outbox worker,
// published to Redis, fanned out by the API's WebSocket handler, and received
// by a subscribed client. It expects the API and worker to be running against
// the same database; see scripts/bench-latency.sh.
package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"net/url"
	"os"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"time"

	"investment_monitoring/config"
	token_service "investment_monitoring/internal/module/support/token/service"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"nhooyr.io/websocket"
)

type envelope struct {
	Type  string `json:"type"`
	Topic string `json:"topic"`
	Data  struct {
		OutboxEventID uuid.UUID `json:"outboxEventId"`
		Subscribed    []string  `json:"subscribed"`
	} `json:"data"`
}

type sample struct {
	id                  uuid.UUID
	committed, received time.Time
	commitToProcessedMs float64
	status              string
}

func main() {
	n := flag.Int("n", 1000, "measured events")
	warmup := flag.Int("warmup", 20, "events sent before measuring")
	rate := flag.Float64("rate", 20, "events per second")
	wsURL := flag.String("ws", "ws://localhost:7777/api/realtime/ws/", "WebSocket endpoint")
	label := flag.String("label", "listen", "label for this run")
	out := flag.String("out", "", "CSV file for raw samples")
	wait := flag.Duration("wait", 30*time.Second, "how long to wait for outstanding events at the end")
	flag.Parse()

	cfg := config.Load()
	db, err := config.ConnectDB(cfg.DATABASE_URL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()

	// Subscribe as an admin of the lender organization of a dedicated project,
	// so the client passes the same topic authorization a real user does.
	userID, projectID := uuid.New(), uuid.New()
	email := userID.String() + "@bench.local"
	govID, lenderID := uuid.New(), uuid.New()
	for _, stmt := range []struct {
		query string
		args  []any
	}{
		{`INSERT INTO organizations (id, name, role) VALUES ($1, 'Bench Supervisor', 'GOV_SUPERVISOR'), ($2, 'Bench Lender', 'LOAN_PROVIDER')`, []any{govID, lenderID}},
		{`INSERT INTO users (id, name, email) VALUES ($1, 'Bench Subscriber', $2)`, []any{userID, email}},
		{`INSERT INTO organization_members (role, user_id, organization_id) VALUES ('ADMIN', $1, $2)`, []any{userID, lenderID}},
		{`INSERT INTO projects (id, name, description, contract_file_url) VALUES ($1, 'Bench Project', 'latency benchmark', 'https://example.test/contract.pdf')`, []any{projectID}},
		{`INSERT INTO contracts (description, project_id, gov_supervisor_participant_id, loan_provider_participant_id) VALUES ('bench', $1, $2, $3)`, []any{projectID, govID, lenderID}},
	} {
		if _, err := db.ExecContext(ctx, stmt.query, stmt.args...); err != nil {
			log.Fatalf("create benchmark fixture: %v", err)
		}
	}
	token, err := token_service.NewService(cfg).GenerateAccessToken(token_service.GenerateAccessTokenInput{ID: userID, Email: email})
	if err != nil {
		log.Fatal(err)
	}

	conn, _, err := websocket.Dial(ctx, *wsURL+"?accessToken="+url.QueryEscape(token), nil)
	if err != nil {
		log.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "done")
	conn.SetReadLimit(1 << 20)

	topic := fmt.Sprintf("project.%s.finance", projectID)
	sub, _ := json.Marshal(map[string]string{"type": "subscribe", "topic": topic})
	if err := conn.Write(ctx, websocket.MessageText, sub); err != nil {
		log.Fatalf("subscribe: %v", err)
	}
	if err := awaitSubscription(ctx, conn, topic); err != nil {
		log.Fatal(err)
	}

	// Heartbeat like the dashboard client does; the server drops connections
	// that stay silent longer than WEBSOCKET_IDLE_TIMEOUT_SECONDS.
	go func() {
		ping, _ := json.Marshal(map[string]string{"type": "ping"})
		for range time.Tick(cfg.WEBSOCKET_PING_INTERVAL) {
			if conn.Write(ctx, websocket.MessageText, ping) != nil {
				return
			}
		}
	}()

	var mu sync.Mutex
	received := map[uuid.UUID]time.Time{}
	go func() {
		for {
			_, data, err := conn.Read(ctx)
			if err != nil {
				return
			}
			at := time.Now()
			var env envelope
			if json.Unmarshal(data, &env) != nil || env.Topic != topic {
				continue
			}
			mu.Lock()
			if _, seen := received[env.Data.OutboxEventID]; !seen {
				received[env.Data.OutboxEventID] = at
			}
			mu.Unlock()
		}
	}()

	total := *warmup + *n
	interval := time.Duration(float64(time.Second) / *rate)
	samples := make([]sample, 0, total)
	for i := 0; i < total; i++ {
		id := uuid.New()
		payload, _ := json.Marshal(map[string]any{"projectId": projectID, "paymentRequestId": uuid.New(), "bench": true})
		if _, err := db.ExecContext(ctx,
			`INSERT INTO outbox_events (id, event_type, aggregate_type, aggregate_id, payload) VALUES ($1, 'PaymentPaid', 'payment_request', $2, $3::jsonb)`,
			id, uuid.New(), string(payload)); err != nil {
			log.Fatalf("insert event: %v", err)
		}
		samples = append(samples, sample{id: id, committed: time.Now()})
		time.Sleep(interval)
	}

	deadline := time.Now().Add(*wait)
	for time.Now().Before(deadline) {
		mu.Lock()
		done := len(received) >= total
		mu.Unlock()
		if done {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	ids := make([]uuid.UUID, len(samples))
	for i, s := range samples {
		ids[i] = s.id
	}
	rows, err := db.QueryContext(ctx, `
		SELECT id, status, COALESCE(EXTRACT(EPOCH FROM (processed_at - created_at)) * 1000, -1)
		FROM outbox_events WHERE id = ANY($1)`, pq.Array(ids))
	if err != nil {
		log.Fatal(err)
	}
	type dbRow struct {
		status string
		ms     float64
	}
	byID := map[uuid.UUID]dbRow{}
	for rows.Next() {
		var id uuid.UUID
		var r dbRow
		if err := rows.Scan(&id, &r.status, &r.ms); err != nil {
			log.Fatal(err)
		}
		byID[id] = r
	}
	rows.Close()

	mu.Lock()
	for i := range samples {
		samples[i].received = received[samples[i].id]
		samples[i].status = byID[samples[i].id].status
		samples[i].commitToProcessedMs = byID[samples[i].id].ms
	}
	mu.Unlock()
	measured := samples[*warmup:]

	var e2e, worker []float64
	lost := 0
	for _, s := range measured {
		if s.received.IsZero() {
			lost++
			continue
		}
		e2e = append(e2e, float64(s.received.Sub(s.committed).Microseconds())/1000)
		if s.commitToProcessedMs >= 0 {
			worker = append(worker, s.commitToProcessedMs)
		}
	}

	var pgVersion string
	_ = db.QueryRowContext(ctx, `SHOW server_version`).Scan(&pgVersion)
	fmt.Printf("run=%s n=%d warmup=%d rate=%.1f/s lost=%d go=%s os=%s/%s cpus=%d postgres=%s\n",
		*label, len(measured), *warmup, *rate, lost, runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.NumCPU(), pgVersion)
	printStats("commit_to_client_ms   ", e2e)
	printStats("insert_to_processed_ms", worker)

	if *out != "" {
		if err := writeCSV(*out, *label, measured); err != nil {
			log.Fatal(err)
		}
		fmt.Println("samples written to", *out)
	}
}

func awaitSubscription(ctx context.Context, conn *websocket.Conn, topic string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return fmt.Errorf("await subscription: %w", err)
		}
		var env envelope
		if json.Unmarshal(data, &env) != nil {
			continue
		}
		for _, t := range env.Data.Subscribed {
			if t == topic {
				return nil
			}
		}
	}
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return math.NaN()
	}
	rank := int(math.Ceil(p/100*float64(len(sorted)))) - 1
	if rank < 0 {
		rank = 0
	}
	return sorted[rank]
}

func printStats(name string, values []float64) {
	if len(values) == 0 {
		fmt.Printf("%s no samples\n", name)
		return
	}
	sort.Float64s(values)
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	fmt.Printf("%s samples=%d mean=%.1f p50=%.1f p95=%.1f p99=%.1f max=%.1f\n",
		name, len(values), sum/float64(len(values)),
		percentile(values, 50), percentile(values, 95), percentile(values, 99), values[len(values)-1])
}

func writeCSV(path, label string, samples []sample) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	_ = w.Write([]string{"run", "seq", "event_id", "status", "commit_to_client_ms", "insert_to_processed_ms"})
	for i, s := range samples {
		e2e := ""
		if !s.received.IsZero() {
			e2e = strconv.FormatFloat(float64(s.received.Sub(s.committed).Microseconds())/1000, 'f', 3, 64)
		}
		_ = w.Write([]string{label, strconv.Itoa(i + 1), s.id.String(), s.status, e2e, strconv.FormatFloat(s.commitToProcessedMs, 'f', 3, 64)})
	}
	w.Flush()
	return w.Error()
}
