package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// =====================
// Public API (hooks)
// =====================

// MetricsConfig controls the metrics server.
type MetricsConfig struct {
	Addr              string // e.g. "0.0.0.0:9090"
	Path              string // default "/metrics"
	IncludeUserLabel   bool   // caution: can be high-cardinality
	Namespace          string // default "sftp"
	Subsystem          string // default "server"
	DisableGoCollector bool
	DisableProcess     bool
}

// DefaultMetricsConfigFromEnv reads config from env vars.
// METRICS_ADDR default: "0.0.0.0:9090"
// METRICS_PATH default: "/metrics"
// METRICS_INCLUDE_USER default: "false"
func DefaultMetricsConfigFromEnv() MetricsConfig {
	addr := getenvM("METRICS_ADDR", "0.0.0.0:9090")
	path := getenvM("METRICS_PATH", "/metrics")
	includeUser := getenvBoolM("METRICS_INCLUDE_USER", false)

	ns := getenvM("METRICS_NAMESPACE", "sftp")
	sub := getenvM("METRICS_SUBSYSTEM", "server")

	disableGo := getenvBoolM("METRICS_DISABLE_GO", false)
	disableProc := getenvBoolM("METRICS_DISABLE_PROCESS", false)

	return MetricsConfig{
		Addr:              addr,
		Path:              path,
		IncludeUserLabel:   includeUser,
		Namespace:          ns,
		Subsystem:          sub,
		DisableGoCollector: disableGo,
		DisableProcess:     disableProc,
	}
}

// StartMetricsServer starts /metrics on a separate listener.
// Call this once during startup, and cancel via ctx.
func StartMetricsServer(ctx context.Context, cfg MetricsConfig) {
	if cfg.Addr == "" {
		// Allow disabling by setting empty addr
		log.Printf("metrics disabled (METRICS_ADDR empty)")
		return
	}
	if cfg.Path == "" {
		cfg.Path = "/metrics"
	}

	reg := prometheus.NewRegistry()

	// Register collectors (optional)
	if !cfg.DisableGoCollector {
		reg.MustRegister(prometheus.NewGoCollector())
	}
	if !cfg.DisableProcess {
		reg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	}

	// Register our SFTP metrics
	m := newSFTPMetrics(cfg, reg)
	setGlobalMetrics(m)

	mux := http.NewServeMux()
	mux.Handle(cfg.Path, promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	go func() {
		log.Printf("metrics listening on %s%s", cfg.Addr, cfg.Path)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("metrics server error: %v", err)
		}
	}()
}

// ---- Hook functions (call these from your auth + fs code) ----

// MetricsAuthResult values (low-cardinality)
const (
	AuthOK          = "ok"
	AuthFailKey     = "fail_key"
	AuthFailUnknown = "fail_unknown_user"
	AuthFailDisabled = "fail_disabled"
	AuthError       = "error"
)

// ObserveAuth records an auth attempt and its latency.
func ObserveAuth(user string, result string, dur time.Duration) {
	m := getGlobalMetrics()
	if m == nil {
		return
	}
	lbl := prometheus.Labels{"result": result}
	if m.includeUser {
		lbl["user"] = safeUserLabel(user)
	}
	m.authAttempts.With(lbl).Inc()
	m.authDuration.With(lbl).Observe(dur.Seconds())
}

// IncSessionActive increments/decrements active session gauge.
// Call with +1 on session start, -1 on session end.
func IncSessionActive(delta float64) {
	m := getGlobalMetrics()
	if m == nil {
		return
	}
	m.sessionsActive.Add(delta)
}

// IncSessionTotal increments session total count.
func IncSessionTotal(result string) {
	m := getGlobalMetrics()
	if m == nil {
		return
	}
	m.sessionsTotal.WithLabelValues(result).Inc()
}

// ObserveOp records an SFTP operation duration and outcome.
// op examples: "ls","get","put","rm","rename","mkdir","rmdir","stat"
func ObserveOp(user string, op string, result string, dur time.Duration) {
	m := getGlobalMetrics()
	if m == nil {
		return
	}
	lbl := prometheus.Labels{
		"op":     normalizeOpM(op),
		"result": result,
	}
	if m.includeUser {
		lbl["user"] = safeUserLabel(user)
	}
	m.opTotal.With(lbl).Inc()
	m.opDuration.With(lbl).Observe(dur.Seconds())
}

// AddBytesIn adds uploaded bytes.
func AddBytesIn(user string, result string, n int64) {
	m := getGlobalMetrics()
	if m == nil || n <= 0 {
		return
	}
	lbl := prometheus.Labels{"result": result}
	if m.includeUser {
		lbl["user"] = safeUserLabel(user)
	}
	m.bytesIn.With(lbl).Add(float64(n))
}

// AddBytesOut adds downloaded bytes.
func AddBytesOut(user string, result string, n int64) {
	m := getGlobalMetrics()
	if m == nil || n <= 0 {
		return
	}
	lbl := prometheus.Labels{"result": result}
	if m.includeUser {
		lbl["user"] = safeUserLabel(user)
	}
	m.bytesOut.With(lbl).Add(float64(n))
}

// IncQuotaExceeded increments quota exceeded counter.
func IncQuotaExceeded(user string, quotaType string) {
	m := getGlobalMetrics()
	if m == nil {
		return
	}
	lbl := prometheus.Labels{"type": quotaType}
	if m.includeUser {
		lbl["user"] = safeUserLabel(user)
	}
	m.quotaExceeded.With(lbl).Inc()
}

// ObserveVault records Vault request stats.
func ObserveVault(op string, result string, dur time.Duration) {
	m := getGlobalMetrics()
	if m == nil {
		return
	}
	m.vaultReqs.WithLabelValues(op, result).Inc()
	m.vaultDuration.WithLabelValues(op, result).Observe(dur.Seconds())
	if result == "success" {
		m.vaultLastSuccess.Set(float64(time.Now().Unix()))
	}
}

// IncStorageIOError increments storage IO error counter.
func IncStorageIOError(op string) {
	m := getGlobalMetrics()
	if m == nil {
		return
	}
	m.storageIOErrors.WithLabelValues(op).Inc()
}

// =====================
// Internal metrics impl
// =====================

type sftpMetrics struct {
	includeUser bool

	sessionsActive prometheus.Gauge
	sessionsTotal  *prometheus.CounterVec

	authAttempts *prometheus.CounterVec
	authDuration *prometheus.HistogramVec

	opTotal    *prometheus.CounterVec
	opDuration *prometheus.HistogramVec

	bytesIn  *prometheus.CounterVec
	bytesOut *prometheus.CounterVec

	quotaExceeded *prometheus.CounterVec

	vaultReqs        *prometheus.CounterVec
	vaultDuration    *prometheus.HistogramVec
	vaultLastSuccess prometheus.Gauge

	storageIOErrors *prometheus.CounterVec
}

func newSFTPMetrics(cfg MetricsConfig, reg *prometheus.Registry) *sftpMetrics {
	ns := cfg.Namespace
	sub := cfg.Subsystem
	includeUser := cfg.IncludeUserLabel

	// dynamic labels: keep low-card by default
	authLabels := []string{"result"}
	opLabels := []string{"op", "result"}
	byteLabels := []string{"result"}
	quotaLabels := []string{"type"}

	if includeUser {
		authLabels = append(authLabels, "user")
		opLabels = append(opLabels, "user")
		byteLabels = append(byteLabels, "user")
		quotaLabels = append(quotaLabels, "user")
	}

	m := &sftpMetrics{includeUser: includeUser}

	m.sessionsActive = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: ns, Subsystem: sub, Name: "sessions_active",
		Help: "Current number of active SFTP sessions.",
	})
	m.sessionsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: ns, Subsystem: sub, Name: "sessions_total",
		Help: "Total number of SFTP sessions started.",
	}, []string{"result"})

	m.authAttempts = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: ns, Subsystem: sub, Name: "auth_attempts_total",
		Help: "Total authentication attempts.",
	}, authLabels)
	m.authDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: ns, Subsystem: sub, Name: "auth_duration_seconds",
		Help: "Authentication decision latency.",
		Buckets: []float64{0.002, 0.005, 0.01, 0.02, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
	}, authLabels)

	m.opTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: ns, Subsystem: sub, Name: "ops_total",
		Help: "Total SFTP operations.",
	}, opLabels)
	m.opDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: ns, Subsystem: sub, Name: "op_duration_seconds",
		Help: "SFTP operation latency.",
		Buckets: []float64{0.002, 0.005, 0.01, 0.02, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10, 30},
	}, opLabels)

	m.bytesIn = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: ns, Subsystem: sub, Name: "bytes_in_total",
		Help: "Total bytes uploaded to the server.",
	}, byteLabels)
	m.bytesOut = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: ns, Subsystem: sub, Name: "bytes_out_total",
		Help: "Total bytes downloaded from the server.",
	}, byteLabels)

	m.quotaExceeded = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: ns, Subsystem: sub, Name: "quota_exceeded_total",
		Help: "Total quota exceed events.",
	}, quotaLabels)

	m.vaultReqs = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: ns, Subsystem: sub, Name: "vault_requests_total",
		Help: "Total Vault API requests.",
	}, []string{"op", "result"})
	m.vaultDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: ns, Subsystem: sub, Name: "vault_request_duration_seconds",
		Help: "Vault API request latency.",
		Buckets: []float64{0.005, 0.01, 0.02, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
	}, []string{"op", "result"})
	m.vaultLastSuccess = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: ns, Subsystem: sub, Name: "vault_last_success_timestamp_seconds",
		Help: "Unix timestamp of last successful Vault request.",
	})

	m.storageIOErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: ns, Subsystem: sub, Name: "storage_io_errors_total",
		Help: "Storage IO error count (application-level).",
	}, []string{"op"})

	reg.MustRegister(
		m.sessionsActive,
		m.sessionsTotal,
		m.authAttempts,
		m.authDuration,
		m.opTotal,
		m.opDuration,
		m.bytesIn,
		m.bytesOut,
		m.quotaExceeded,
		m.vaultReqs,
		m.vaultDuration,
		m.vaultLastSuccess,
		m.storageIOErrors,
	)

	return m
}

var globalMetrics *sftpMetrics

func setGlobalMetrics(m *sftpMetrics) { globalMetrics = m }
func getGlobalMetrics() *sftpMetrics  { return globalMetrics }

// =====================
// Helpers
// =====================


func getenvM(k, def string) string {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	return v
}

func getenvBoolM(k string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func normalizeOpM(op string) string {
	op = strings.ToLower(strings.TrimSpace(op))
	switch op {
	case "list", "readdir":
		return "ls"
	case "read", "get":
		return "get"
	case "write", "put":
		return "put"
	default:
		return op
	}
}

// safeUserLabel avoids empty label values; you can also hash usernames here if desired.
func safeUserLabel(u string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return "unknown"
	}
	// keep it short-ish; optional
	if len(u) > 64 {
		return u[:64]
	}
	return u
}

