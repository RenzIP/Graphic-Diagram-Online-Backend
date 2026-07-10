// Package p is the Google Cloud Functions (Gen 2) entry point for the
// GraDiOl API. It initialises the full Fiber application once on cold
// start and exposes an HTTP handler compatible with the Cloud Functions
// runtime.
//
// Deploy with:
//
//	gcloud functions deploy gradiol-api \
//	  --entry-point=Handle \
//	  --trigger-http \
//	  --gen2 \
//	  --runtime=go125 \
//	  --region=asia-southeast2 \
//	  --allow-unauthenticated
//
// Set the following secrets via --set-env-vars or the GCP console:
//
//	ENV=production
//	JWT_SECRET=<your-secret>
//	SUPABASE_DATABASE_URL=<your-database-url>
//	FRONTEND_URL=<your-frontend-url>
//	BACKEND_URL=<your-cloud-run-url>
//	REDIS_URL=<your-redis-url>
//	GOOGLE_CLIENT_ID=<your-google-client-id>
//	GOOGLE_CLIENT_SECRET=<your-google-client-secret>
//	GITHUB_CLIENT_ID=<your-github-client-id>
//	GITHUB_CLIENT_SECRET=<your-github-client-secret>
package p

import (
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"sync"

	"github.com/RenzIP/Graphic-Diagram-Online/app"
	"github.com/valyala/fasthttp"
)

var (
	instance *app.Instance
	once     sync.Once
)

func init() {
	once.Do(func() {
		instance = app.New()
		log.Println("GCF cold start complete — Fiber app initialized")
	})
}

// Handle is the GCF Gen 2 HTTP entry point. It adapts the standard
// net/http request to fasthttp (which Fiber v2 uses internally) and
// copies the response back to the http.ResponseWriter.
func Handle(w http.ResponseWriter, r *http.Request) {
	ctx := &fasthttp.RequestCtx{}

	// --- Convert net/http → fasthttp ---
	ctx.Request.SetRequestURI(r.URL.RequestURI())
	ctx.Request.Header.SetMethod(r.Method)
	ctx.Request.SetHost(r.Host)

	// Copy client IP (needed for Fiber logger, rate limiter, c.IP())
	if host, port, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		if ip := net.ParseIP(host); ip != nil {
			if portNum, portErr := strconv.Atoi(port); portErr == nil {
				ctx.SetRemoteAddr(&net.TCPAddr{IP: ip, Port: portNum})
			}
		}
	}

	// Copy request headers
	for key, values := range r.Header {
		for _, v := range values {
			ctx.Request.Header.Add(key, v)
		}
	}

	// Determine scheme — GCF Gen 2 terminates TLS at the Google Front
	// End, so r.TLS is nil and r.URL.Scheme is empty. Check
	// X-Forwarded-Proto first, then fall back to r.TLS presence.
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		ctx.Request.URI().SetScheme(proto)
	} else if scheme := r.URL.Scheme; scheme != "" {
		ctx.Request.URI().SetScheme(scheme)
	} else if r.TLS != nil {
		ctx.Request.URI().SetScheme("https")
	} else {
		ctx.Request.URI().SetScheme("http")
	}

	// Copy request body
	if r.Body != nil {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read request body", http.StatusBadRequest)
			return
		}
		r.Body.Close()
		ctx.Request.SetBody(body)
	}

	// --- Serve via Fiber's internal fasthttp handler ---
	instance.App.Handler()(ctx)

	// --- Convert fasthttp → net/http response ---
	ctx.Response.Header.VisitAll(func(key, value []byte) {
		w.Header().Set(string(key), string(value))
	})
	w.WriteHeader(ctx.Response.StatusCode())
	if body := ctx.Response.Body(); len(body) > 0 {
		w.Write(body)
	}
}
