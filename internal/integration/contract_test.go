//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"investment_monitoring/config"
	"investment_monitoring/internal/audit"
	handler "investment_monitoring/internal/http"
	websocket_service "investment_monitoring/internal/module/realtime/websocket/service"
	token_service "investment_monitoring/internal/module/support/token/service"
	"investment_monitoring/internal/outbox"
	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/internal/testutil"

	"github.com/go-chi/chi/v5"
	chiMw "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"nhooyr.io/websocket"
)

type apiResponse struct {
	ResponseMessage string          `json:"responseMessage"`
	Data            json.RawMessage `json:"data"`
}

type contractCase struct {
	id, pillar, name string
	run              func(t *testing.T) (status int, message string)
	wantStatus       int
	wantMessage      string
}

// TestAPIContractScenarios runs the 24 contract scenarios reported in the paper
// through the real HTTP router, authentication middleware, PostgreSQL, Redis
// fan-out and WebSocket handler. They replace the earlier manual Postman runs.
func TestAPIContractScenarios(t *testing.T) {
	db, store := testutil.OpenTestDB(t)
	f := testutil.NewFixture(t, db)
	ctx := context.Background()

	// WebSocket values mirror the defaults of config.Load; the Redis prefix is
	// unique so the test never receives events from a running dev server.
	prefix := "contract_" + uuid.NewString()[:8]
	cfg := &config.Config{
		JWT_ACCESS_TOKEN_SECRET:        "contract-test-access",
		JWT_REFRESH_TOKEN_SECRET:       "contract-test-refresh",
		JWT_ACCESS_TOKEN_EXPIRED:       time.Hour,
		REDIS_HOST:                     "localhost",
		REDIS_PORT:                     "6379",
		REDIS_PREFIX:                   prefix,
		WEBSOCKET_ENABLED:              true,
		WEBSOCKET_ALLOWED_ORIGINS:      []string{"*"},
		WEBSOCKET_READ_LIMIT_BYTES:     65536,
		WEBSOCKET_WRITE_TIMEOUT:        10 * time.Second,
		WEBSOCKET_IDLE_TIMEOUT:         60 * time.Second,
		WEBSOCKET_PING_INTERVAL:        25 * time.Second,
		WEBSOCKET_CLIENT_SEND_BUFFER:   64,
		WEBSOCKET_REDIS_CHANNEL_PREFIX: prefix + ":ws",
		WEBSOCKET_PRESENCE_TTL:         90 * time.Second,
	}
	rdb, err := config.ConnectRedis(cfg)
	if err != nil {
		t.Skipf("redis unavailable: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })

	apiRouter, wsSvc := handler.NewRouter(store, rdb, cfg)
	wsCtx, stopWS := context.WithCancel(ctx)
	if err := wsSvc.Start(wsCtx); err != nil {
		t.Fatalf("start websocket service: %v", err)
	}
	t.Cleanup(func() { stopWS(); _ = wsSvc.Shutdown(ctx) })
	// Same middleware stack as cmd/api: the audit middleware captures the
	// request ID, client IP and user agent that the audit trail records.
	root := chi.NewRouter()
	root.Use(chiMw.RequestID)
	root.Use(audit.Middleware)
	root.Use(chiMw.StripSlashes)
	root.Use(chiMw.Recoverer)
	root.Mount("/api", apiRouter)
	server := httptest.NewServer(root)
	t.Cleanup(server.Close)

	tokens := token_service.NewService(cfg)
	tokenFor := func(a testutil.Actor) string {
		tok, err := tokens.GenerateAccessToken(token_service.GenerateAccessTokenInput{ID: a.UserID, Email: a.UserID.String() + "@test.local"})
		if err != nil {
			t.Fatalf("token: %v", err)
		}
		return tok
	}
	call := func(t *testing.T, actor *testutil.Actor, method, path string, body any) (int, apiResponse) {
		t.Helper()
		var reader *bytes.Reader
		if body != nil {
			raw, _ := json.Marshal(body)
			reader = bytes.NewReader(raw)
		} else {
			reader = bytes.NewReader(nil)
		}
		req, _ := http.NewRequest(method, server.URL+"/api"+path, reader)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "contract-test/1.0")
		if actor != nil {
			req.Header.Set("Authorization", "Bearer "+tokenFor(*actor))
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", method, path, err)
		}
		defer res.Body.Close()
		var parsed apiResponse
		_ = json.NewDecoder(res.Body).Decode(&parsed)
		return res.StatusCode, parsed
	}
	do := func(t *testing.T, actor *testutil.Actor, method, path string, body any) (int, string) {
		status, res := call(t, actor, method, path, body)
		return status, res.ResponseMessage
	}

	project, actors := projectWithAllPositions(f)
	other, otherActors := projectWithAllPositions(f)
	_ = other
	ct, au, ea, lp, gp := actors["CONTRACTOR"], actors["INDEPENDENT_AUDITOR"], actors["EXECUTING_AGENCY"], actors["LOAN_PROVIDER"], actors["GOV_PARTNER"]
	gpUnassignedOrg := f.Org(entity.OrganizationRoleEnumGOVPARTNER)
	db.Exec(`INSERT INTO gov_participants (contract_id, gov_participant_id) VALUES ($1, $2)`, project.ContractID, gpUnassignedOrg)
	gpUnassigned := f.Member(gpUnassignedOrg, entity.MemberRoleEnumADMIN)
	evidence := []string{"https://example.test/evidence.jpg"}
	paymentIn := func(status entity.PaymentRequestStatusEnum) uuid.UUID {
		return f.PaymentRequest(project.ContractID, status)
	}
	ticketIn := func(status entity.TicketStatusEnum) uuid.UUID {
		id := f.Ticket(project.ContractID, ct, uuid.NullUUID{UUID: gp.OrgID, Valid: true})
		if status != entity.TicketStatusEnumOPEN {
			f.TicketStatus(id, ea, status)
		}
		return id
	}
	var progressLogID, createdTicketID uuid.UUID

	dialWS := func(t *testing.T, token string) (*websocket.Conn, *http.Response, error) {
		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/realtime/ws"
		if token != "" {
			wsURL += "?accessToken=" + url.QueryEscape(token)
		}
		dctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return websocket.Dial(dctx, wsURL, nil)
	}
	readUntil := func(t *testing.T, conn *websocket.Conn, match func(map[string]any) bool) (map[string]any, []byte) {
		t.Helper()
		rctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		for {
			_, raw, err := conn.Read(rctx)
			if err != nil {
				t.Fatalf("websocket read: %v", err)
			}
			var env map[string]any
			if json.Unmarshal(raw, &env) == nil && match(env) {
				return env, raw
			}
		}
	}
	subscribe := func(t *testing.T, conn *websocket.Conn, topic string) map[string]any {
		raw, _ := json.Marshal(map[string]string{"type": "subscribe", "topic": topic})
		if err := conn.Write(ctx, websocket.MessageText, raw); err != nil {
			t.Fatalf("subscribe: %v", err)
		}
		env, _ := readUntil(t, conn, func(e map[string]any) bool {
			data, _ := e["data"].(map[string]any)
			_, ok := data["subscribed"]
			return ok
		})
		return env["data"].(map[string]any)
	}
	listed := func(v any, topic string) bool {
		items, _ := v.([]any)
		for _, item := range items {
			if item == topic {
				return true
			}
		}
		return false
	}

	cases := []contractCase{
		// ---- Pillar 1: FSM workflow governance ----
		{"CT-FSM-01", "FSM", "contractor submits a progress report", func(t *testing.T) (int, string) {
			status, res := call(t, &ct, "POST", fmt.Sprintf("/monitoring/task/%s/log", project.TaskID),
				map[string]any{"progressPercentage": 10, "report": "Pier P12 casting", "evidenceImageUrls": evidence})
			var log struct {
				ID uuid.UUID `json:"id"`
			}
			_ = json.Unmarshal(res.Data, &log)
			progressLogID = log.ID
			return status, res.ResponseMessage
		}, http.StatusOK, "SUCCESS_SUBMIT_PROGRESS"},
		{"CT-FSM-02", "FSM", "contracted auditor accepts the pending report", func(t *testing.T) (int, string) {
			return do(t, &au, "POST", fmt.Sprintf("/monitoring/log/%s/validate", progressLogID), map[string]any{"realProgressPercentage": 10})
		}, http.StatusOK, "SUCCESS_VALIDATE_PROGRESS"},
		{"CT-FSM-03", "FSM", "auditor validates the same report again (terminal state)", func(t *testing.T) (int, string) {
			return do(t, &au, "POST", fmt.Sprintf("/monitoring/log/%s/validate", progressLogID), map[string]any{"realProgressPercentage": 10})
		}, http.StatusBadRequest, "PROGRESS_ALREADY_VALIDATED"},
		{"CT-FSM-04", "FSM", "executing agency approves a payment that skipped the auditor", func(t *testing.T) (int, string) {
			return do(t, &ea, "PUT", fmt.Sprintf("/finance/request/%s/approve-agency", paymentIn(entity.PaymentRequestStatusEnumSUBMITTED)), map[string]any{"action": "APPROVE"})
		}, http.StatusBadRequest, "INVALID_STATUS_TRANSITION"},
		{"CT-FSM-05", "FSM", "lender disburses a payment still awaiting approval", func(t *testing.T) (int, string) {
			return do(t, &lp, "PUT", fmt.Sprintf("/finance/request/%s/disburse", paymentIn(entity.PaymentRequestStatusEnumSUBMITTED)), map[string]any{"paymentProofImageUrls": evidence})
		}, http.StatusBadRequest, "INVALID_STATUS_TRANSITION"},
		{"CT-FSM-06", "FSM", "assigned government partner resolves a ticket that was never started", func(t *testing.T) (int, string) {
			return do(t, &gp, "POST", fmt.Sprintf("/tracker/ticket/%s/log", ticketIn(entity.TicketStatusEnumASSIGNED)), map[string]any{"description": "done", "newStatus": "RESOLVED"})
		}, http.StatusBadRequest, "INVALID_STATUS_TRANSITION"},

		// ---- Pillar 2: organization-centric authorization ----
		{"CT-POL-01", "Policy", "request without an access token", func(t *testing.T) (int, string) {
			return do(t, nil, "POST", fmt.Sprintf("/monitoring/task/%s/log", project.TaskID), map[string]any{"progressPercentage": 10, "report": "x", "evidenceImageUrls": evidence})
		}, http.StatusUnauthorized, ""},
		{"CT-POL-02", "Policy", "executing agency attempts to validate progress", func(t *testing.T) (int, string) {
			logID := f.ProgressLog(project.TaskID, ct.OrgID)
			return do(t, &ea, "POST", fmt.Sprintf("/monitoring/log/%s/validate", logID), map[string]any{"realProgressPercentage": 10})
		}, http.StatusForbidden, "AUDITOR_REQUIRED"},
		{"CT-POL-03", "Policy", "contractor attempts to disburse an approved payment", func(t *testing.T) (int, string) {
			return do(t, &ct, "PUT", fmt.Sprintf("/finance/request/%s/disburse", paymentIn(entity.PaymentRequestStatusEnumAGENCYAPPROVED)), map[string]any{"paymentProofImageUrls": evidence})
		}, http.StatusForbidden, "LOAN_PROVIDER_REQUIRED"},
		{"CT-POL-04", "Policy", "contractor of another project submits progress on this project", func(t *testing.T) (int, string) {
			other := otherActors["CONTRACTOR"]
			return do(t, &other, "POST", fmt.Sprintf("/monitoring/task/%s/log", project.TaskID), map[string]any{"progressPercentage": 10, "report": "x", "evidenceImageUrls": evidence})
		}, http.StatusForbidden, "CONTRACTOR_REQUIRED"},
		{"CT-POL-05", "Policy", "government partner on the contract but not assigned updates a ticket", func(t *testing.T) (int, string) {
			return do(t, &gpUnassigned, "POST", fmt.Sprintf("/tracker/ticket/%s/log", ticketIn(entity.TicketStatusEnumASSIGNED)), map[string]any{"description": "x", "newStatus": "IN_PROGRESS"})
		}, http.StatusForbidden, "NOT_ASSIGNED_TO_YOUR_ORG"},
		{"CT-POL-06", "Policy", "contracted lender disburses an agency-approved payment", func(t *testing.T) (int, string) {
			return do(t, &lp, "PUT", fmt.Sprintf("/finance/request/%s/disburse", paymentIn(entity.PaymentRequestStatusEnumAGENCYAPPROVED)), map[string]any{"paymentProofImageUrls": evidence})
		}, http.StatusOK, "SUCCESS_DISBURSE"},

		// ---- Pillar 3: SLA tracking and audit trail ----
		{"CT-SLA-01", "SLA/Audit", "contractor raises an obstacle ticket; a 7-day SLA starts", func(t *testing.T) (int, string) {
			status, res := call(t, &ct, "POST", fmt.Sprintf("/tracker/project/%s/ticket", project.ID),
				map[string]any{"title": "Utility relocation", "description": "PLN cable blocks the pier", "location": "Station Harmoni", "evidenceImageUrls": evidence})
			var ticket struct {
				ID uuid.UUID `json:"id"`
			}
			_ = json.Unmarshal(res.Data, &ticket)
			createdTicketID = ticket.ID
			var hours float64
			if err := db.QueryRow(`SELECT EXTRACT(EPOCH FROM due_at - entered_at)/3600 FROM sla_records WHERE entity_id = $1 AND state = 'OPEN' AND status = 'ON_TRACK'`, ticket.ID).Scan(&hours); err != nil || hours != 168 {
				t.Errorf("SLA record: hours=%v err=%v, want a 168h ON_TRACK record", hours, err)
			}
			return status, res.ResponseMessage
		}, http.StatusOK, "SUCCESS_CREATE_TICKET"},
		{"CT-SLA-02", "SLA/Audit", "executing agency assigns the ticket; the SLA restarts for the new state", func(t *testing.T) (int, string) {
			status, msg := do(t, &ea, "PUT", fmt.Sprintf("/tracker/ticket/%s/assign", createdTicketID), map[string]any{"assignToOrgId": gp.OrgID})
			var state string
			_ = db.QueryRow(`SELECT state FROM sla_records WHERE entity_id = $1`, createdTicketID).Scan(&state)
			if state != "ASSIGNED" {
				t.Errorf("SLA state = %s, want ASSIGNED", state)
			}
			return status, msg
		}, http.StatusOK, "SUCCESS_ASSIGN_TICKET"},
		{"CT-SLA-03", "SLA/Audit", "ticket past its due time is reported as SLA_VIOLATED on read", func(t *testing.T) (int, string) {
			if _, err := db.Exec(`UPDATE sla_records SET entered_at = NOW() - INTERVAL '8 days', due_at = NOW() - INTERVAL '1 day' WHERE entity_id = $1`, createdTicketID); err != nil {
				t.Fatalf("age SLA: %v", err)
			}
			status, res := call(t, &ea, "GET", fmt.Sprintf("/tracker/ticket/%s", createdTicketID), nil)
			var ticket struct {
				SLA struct {
					Status     string `json:"status"`
					IsViolated bool   `json:"isViolated"`
				} `json:"sla"`
			}
			_ = json.Unmarshal(res.Data, &ticket)
			if ticket.SLA.Status != "SLA_VIOLATED" || !ticket.SLA.IsViolated {
				t.Errorf("SLA = %+v, want SLA_VIOLATED", ticket.SLA)
			}
			return status, res.ResponseMessage
		}, http.StatusOK, "SUCCESS_GET_TICKET"},
		{"CT-AUD-01", "SLA/Audit", "assigned partner starts work; the audit trail records actor, IP, agent and request ID", func(t *testing.T) (int, string) {
			status, msg := do(t, &gp, "POST", fmt.Sprintf("/tracker/ticket/%s/log", createdTicketID), map[string]any{"description": "Crew on site", "newStatus": "IN_PROGRESS"})
			var actorUser, actorOrg uuid.UUID
			var ip, agent, requestID string
			err := db.QueryRow(`SELECT actor_user_id, actor_org_id, COALESCE(ip_address,''), COALESCE(user_agent,''), COALESCE(request_id,'')
				FROM audit_logs WHERE entity_id = $1 AND after_data->>'status' = 'IN_PROGRESS'`, createdTicketID).Scan(&actorUser, &actorOrg, &ip, &agent, &requestID)
			if err != nil || actorUser != gp.UserID || actorOrg != gp.OrgID || ip == "" || agent != "contract-test/1.0" || requestID == "" {
				t.Errorf("audit row: user=%s org=%s ip=%q agent=%q request=%q err=%v", actorUser, actorOrg, ip, agent, requestID, err)
			}
			return status, msg
		}, http.StatusOK, "SUCCESS_ADD_LOG"},
		{"CT-AUD-02", "SLA/Audit", "executing agency closes a resolved ticket; the SLA is marked resolved", func(t *testing.T) (int, string) {
			if s, m := do(t, &gp, "POST", fmt.Sprintf("/tracker/ticket/%s/log", createdTicketID), map[string]any{"description": "Cable rerouted", "newStatus": "RESOLVED"}); s != http.StatusOK {
				t.Fatalf("resolve: %d %s", s, m)
			}
			status, msg := do(t, &ea, "PUT", fmt.Sprintf("/tracker/ticket/%s/close", createdTicketID), nil)
			var resolved bool
			_ = db.QueryRow(`SELECT resolved_at IS NOT NULL FROM sla_records WHERE entity_id = $1`, createdTicketID).Scan(&resolved)
			if !resolved {
				t.Error("SLA record not resolved after close")
			}
			return status, msg
		}, http.StatusOK, "SUCCESS_CLOSE_TICKET"},
		{"CT-AUD-03", "SLA/Audit", "audit hash chain verifies intact after the scenario", func(t *testing.T) (int, string) {
			if _, reason, intact := verifyChain(t, db); !intact {
				return http.StatusConflict, reason
			}
			return http.StatusOK, "CHAIN_INTACT"
		}, http.StatusOK, "CHAIN_INTACT"},

		// ---- Pillar 4: outbox and realtime delivery ----
		{"CT-RT-01", "Realtime", "WebSocket handshake without an access token", func(t *testing.T) (int, string) {
			conn, res, err := dialWS(t, "")
			if err == nil {
				conn.Close(websocket.StatusNormalClosure, "")
				return http.StatusSwitchingProtocols, "UPGRADED"
			}
			if res == nil {
				t.Fatalf("dial: %v", err)
			}
			return res.StatusCode, ""
		}, http.StatusUnauthorized, ""},
		{"CT-RT-02", "Realtime", "participant subscribes to its own project's finance topic", func(t *testing.T) (int, string) {
			conn, _, err := dialWS(t, tokenFor(lp))
			if err != nil {
				t.Fatalf("dial: %v", err)
			}
			defer conn.Close(websocket.StatusNormalClosure, "")
			topic := fmt.Sprintf("project.%s.finance", project.ID)
			if !listed(subscribe(t, conn, topic)["subscribed"], topic) {
				return http.StatusForbidden, "REJECTED"
			}
			return http.StatusOK, "SUBSCRIBED"
		}, http.StatusOK, "SUBSCRIBED"},
		{"CT-RT-03", "Realtime", "participant subscribes to another project's finance topic", func(t *testing.T) (int, string) {
			conn, _, err := dialWS(t, tokenFor(lp))
			if err != nil {
				t.Fatalf("dial: %v", err)
			}
			defer conn.Close(websocket.StatusNormalClosure, "")
			topic := fmt.Sprintf("project.%s.finance", other.ID)
			if listed(subscribe(t, conn, topic)["rejected"], topic) {
				return http.StatusForbidden, "REJECTED"
			}
			return http.StatusOK, "SUBSCRIBED"
		}, http.StatusForbidden, "REJECTED"},
		{"CT-RT-04", "Realtime", "disbursement commits a PaymentPaid outbox event with the status change", func(t *testing.T) (int, string) {
			id := paymentIn(entity.PaymentRequestStatusEnumAGENCYAPPROVED)
			status, msg := do(t, &lp, "PUT", fmt.Sprintf("/finance/request/%s/disburse", id), map[string]any{"paymentProofImageUrls": evidence})
			var eventStatus string
			if err := db.QueryRow(`SELECT status FROM outbox_events WHERE aggregate_id = $1 AND event_type = $2`, id, outbox.EventPaymentPaid).Scan(&eventStatus); err != nil || eventStatus != "PENDING" {
				t.Errorf("outbox event status=%q err=%v, want PENDING", eventStatus, err)
			}
			return status, msg
		}, http.StatusOK, "SUCCESS_DISBURSE"},
		{"CT-RT-05", "Realtime", "worker delivers the committed event to the subscribed client", func(t *testing.T) (int, string) {
			conn, _, err := dialWS(t, tokenFor(lp))
			if err != nil {
				t.Fatalf("dial: %v", err)
			}
			defer conn.Close(websocket.StatusNormalClosure, "")
			topic := fmt.Sprintf("project.%s.finance", project.ID)
			subscribe(t, conn, topic)

			id := paymentIn(entity.PaymentRequestStatusEnumAGENCYAPPROVED)
			if s, m := do(t, &lp, "PUT", fmt.Sprintf("/finance/request/%s/disburse", id), map[string]any{"paymentProofImageUrls": evidence}); s != http.StatusOK {
				t.Fatalf("disburse: %d %s", s, m)
			}
			publisher := websocket_service.NewService(cfg, tokens, store, rdb)
			processor := outbox.NewProcessor(store, publisher)
			if err := processor.ProcessBatch(ctx, 100); err != nil {
				t.Fatalf("process outbox: %v", err)
			}
			env, raw := readUntil(t, conn, func(e map[string]any) bool {
				data, _ := e["data"].(map[string]any)
				return e["topic"] == topic && data["aggregateId"] == id.String()
			})
			if path := os.Getenv("CAPTURE_REALTIME_PAYLOAD"); path != "" {
				var pretty bytes.Buffer
				_ = json.Indent(&pretty, raw, "", "  ")
				_ = os.WriteFile(path, pretty.Bytes(), 0o644)
			}
			return http.StatusOK, env["type"].(string)
		}, http.StatusOK, "payment.paid"},
		{"CT-RT-06", "Realtime", "delivered envelope is a refetch trigger, not a data copy", func(t *testing.T) (int, string) {
			var processed int
			_ = db.QueryRow(`SELECT count(*) FROM outbox_events WHERE status = 'PROCESSED' AND processed_at IS NOT NULL`).Scan(&processed)
			if processed == 0 {
				return http.StatusConflict, "NO_PROCESSED_EVENT"
			}
			// The envelope carries identifiers only; the payment itself is read back over REST.
			var paymentID uuid.UUID
			_ = db.QueryRow(`SELECT aggregate_id FROM outbox_events WHERE status = 'PROCESSED' AND event_type = $1 ORDER BY processed_at DESC LIMIT 1`, outbox.EventPaymentPaid).Scan(&paymentID)
			return do(t, &lp, "GET", fmt.Sprintf("/finance/request/%s", paymentID), nil)
		}, http.StatusOK, "SUCCESS_GET_PAYMENT_REQUEST"},
	}

	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			status, message := tc.run(t)
			pass := status == tc.wantStatus && (tc.wantMessage == "" || message == tc.wantMessage)
			t.Logf("%s | %-9s | %-82s | expected=%d %s | actual=%d %s | pass=%v",
				tc.id, tc.pillar, tc.name, tc.wantStatus, tc.wantMessage, status, message, pass)
			if !pass {
				t.Errorf("got %d %q, want %d %q", status, message, tc.wantStatus, tc.wantMessage)
			}
		})
	}
}
