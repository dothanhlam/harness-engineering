package monitor

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/dothanhlam/harness-engineering/internal/events"
)

//go:embed index.html
var webFS embed.FS

// heartbeat keeps an idle SSE connection alive and lets the server notice a
// vanished client promptly (the write fails), between real events.
const heartbeat = 20 * time.Second

// Serve runs the web dashboard until ctx is cancelled. It binds addr (callers
// pass a loopback address — this is a local dev tool, not a public service) and
// streams the event log at path over Server-Sent Events.
//
// It returns the listen error synchronously if the port is taken, so the caller
// can report it instead of printing a URL that never comes up.
func Serve(ctx context.Context, addr, path string) error {
	srv := &http.Server{
		Handler: newMux(path),
		// Bound header reads so a stalled client cannot hold a connection open.
		// No WriteTimeout: SSE responses are intentionally long-lived.
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Listen synchronously so a bind failure surfaces here, before we claim to
	// be serving.
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve(ln) }()

	select {
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

// newMux builds the dashboard's routes over the given event log path.
func newMux(path string) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/events", eventsHandler(path))
	mux.HandleFunc("/api/state", stateHandler(path))
	return mux
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	page, err := webFS.ReadFile("index.html")
	if err != nil {
		http.Error(w, "index unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(page)
}

// eventsHandler streams the log as SSE: it replays what is already there, then
// follows live for the lifetime of the connection. It deliberately does NOT stop
// at run_finished — the page is long-lived and a later run (which truncates the
// log) must flow into the same stream. The browser resets its view when the
// run_id changes.
func eventsHandler(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		h := w.Header()
		h.Set("Content-Type", "text/event-stream")
		h.Set("Cache-Control", "no-cache")
		h.Set("Connection", "keep-alive")
		flusher.Flush()

		// The tailer runs in its own goroutine and feeds events over a channel so
		// that all writes to w happen here, in the handler goroutine, serialised
		// with the heartbeat. Writing w from two goroutines would race.
		evCh := make(chan events.Event, 128)
		go func() {
			tl := events.NewTailer(path)
			_ = tl.Follow(r.Context(), func(ev events.Event) {
				select {
				case evCh <- ev:
				case <-r.Context().Done():
				}
			})
			close(evCh)
		}()

		ping := time.NewTicker(heartbeat)
		defer ping.Stop()

		for {
			select {
			case ev, ok := <-evCh:
				if !ok {
					return
				}
				if !writeSSE(w, ev) {
					return
				}
				flusher.Flush()
			case <-ping.C:
				if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
					return
				}
				flusher.Flush()
			case <-r.Context().Done():
				return
			}
		}
	}
}

// writeSSE emits one event as an SSE frame. The id is the event seq, which the
// browser also uses to dedup a full replay after an automatic reconnect. It
// reports false if the connection is gone.
func writeSSE(w http.ResponseWriter, ev events.Event) bool {
	// Re-marshal the whole event: Payload is json.RawMessage, so this round-trips
	// to the same JSON the log holds. A single line — Marshal never emits a raw
	// newline — so one data: field is correct.
	data, err := json.Marshal(ev)
	if err != nil {
		return true // skip a bad event, keep the stream alive
	}
	_, err = fmt.Fprintf(w, "id: %d\ndata: %s\n\n", ev.Seq, data)
	return err == nil
}

// stateHandler returns every event currently in the log as a JSON array, for a
// client that wants a one-shot snapshot rather than the live stream.
func stateHandler(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		all, err := events.ReadAll(path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if all == nil {
			all = []events.Event{}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(all)
	}
}
