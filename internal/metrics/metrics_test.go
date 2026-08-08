package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMiddleware_RecordsRequestAndPassesThroughResponse(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("created"))
	})

	before := testutil.ToFloat64(httpRequestsTotal.WithLabelValues(http.MethodPost, "/widgets", "201"))

	req := httptest.NewRequest(http.MethodPost, "/widgets", nil)
	rr := httptest.NewRecorder()
	Middleware(inner).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status to pass through unaltered, got %d", rr.Code)
	}
	if rr.Body.String() != "created" {
		t.Errorf("expected body to pass through unaltered, got %q", rr.Body.String())
	}

	after := testutil.ToFloat64(httpRequestsTotal.WithLabelValues(http.MethodPost, "/widgets", "201"))
	if after != before+1 {
		t.Errorf("expected http_requests_total{method=POST,path=/widgets,status=201} to increment by 1, got before=%v after=%v", before, after)
	}
}

func TestMiddleware_DefaultsToStatusOKWhenWriteHeaderNeverCalled(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("implicit 200"))
	})

	before := testutil.ToFloat64(httpRequestsTotal.WithLabelValues(http.MethodGet, "/implicit", "200"))
	req := httptest.NewRequest(http.MethodGet, "/implicit", nil)
	Middleware(inner).ServeHTTP(httptest.NewRecorder(), req)
	after := testutil.ToFloat64(httpRequestsTotal.WithLabelValues(http.MethodGet, "/implicit", "200"))

	if after != before+1 {
		t.Errorf("expected an implicit 200 to be recorded, before=%v after=%v", before, after)
	}
}

func TestHandler_ServesPrometheusExpositionFormat(t *testing.T) {
	RecordPostAdded()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "posts_added_total") {
		t.Error("expected the exposition output to include posts_added_total")
	}
}

func TestRecordFunctions_IncrementExpectedMetrics(t *testing.T) {
	beforePosts := testutil.ToFloat64(postsFetched)
	RecordPostsFetched(5)
	if got := testutil.ToFloat64(postsFetched); got != beforePosts+5 {
		t.Errorf("RecordPostsFetched(5): expected +5, got before=%v after=%v", beforePosts, got)
	}

	beforeAdded := testutil.ToFloat64(postsAdded)
	RecordPostAdded()
	if got := testutil.ToFloat64(postsAdded); got != beforeAdded+1 {
		t.Errorf("RecordPostAdded: expected +1, got before=%v after=%v", beforeAdded, got)
	}

	RecordPostsTotal(42)
	if got := testutil.ToFloat64(postsTotal); got != 42 {
		t.Errorf("RecordPostsTotal(42): expected gauge=42, got %v", got)
	}

	RecordAnalysisOperation(10 * time.Millisecond)
	if got := testutil.ToFloat64(analysisOperations); got < 1 {
		t.Errorf("RecordAnalysisOperation: expected the counter to have incremented, got %v", got)
	}

	beforeDB := testutil.ToFloat64(dbOperations.WithLabelValues("get_all", "success"))
	RecordDBOperation("get_all", "success", 5*time.Millisecond)
	if got := testutil.ToFloat64(dbOperations.WithLabelValues("get_all", "success")); got != beforeDB+1 {
		t.Errorf("RecordDBOperation: expected +1, got before=%v after=%v", beforeDB, got)
	}

	beforeEvt := testutil.ToFloat64(postEventsConsumed.WithLabelValues("created"))
	RecordPostEventConsumed("created")
	if got := testutil.ToFloat64(postEventsConsumed.WithLabelValues("created")); got != beforeEvt+1 {
		t.Errorf("RecordPostEventConsumed: expected +1, got before=%v after=%v", beforeEvt, got)
	}

	beforeJob := testutil.ToFloat64(reanalysisJobsProcessed.WithLabelValues("success"))
	RecordReanalysisJobProcessed("success")
	if got := testutil.ToFloat64(reanalysisJobsProcessed.WithLabelValues("success")); got != beforeJob+1 {
		t.Errorf("RecordReanalysisJobProcessed: expected +1, got before=%v after=%v", beforeJob, got)
	}

	beforeNotif := testutil.ToFloat64(notificationsDelivered.WithLabelValues("scheduled_recheck"))
	RecordNotificationDelivered("scheduled_recheck")
	if got := testutil.ToFloat64(notificationsDelivered.WithLabelValues("scheduled_recheck")); got != beforeNotif+1 {
		t.Errorf("RecordNotificationDelivered: expected +1, got before=%v after=%v", beforeNotif, got)
	}
}
