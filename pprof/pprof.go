package pprof

import (
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
)

const (
	// DefaultPrefix url prefix of pprof
	DefaultPrefix = "/debug/pprof"
)

func getPrefix(prefixOptions ...string) string {
	prefix := DefaultPrefix
	if len(prefixOptions) > 0 && len(prefixOptions[0]) > 0 {
		prefix = prefixOptions[0]
	}
	return prefix
}

func Listen(addr string, prefixOptions ...string) {
	prefix := getPrefix(prefixOptions...)

	mux := http.NewServeMux()
	mux.Handle(prefix+"/", http.HandlerFunc(pprof.Index))
	mux.Handle(prefix+"/allocs", http.HandlerFunc(pprof.Handler("allocs").ServeHTTP))
	mux.Handle(prefix+"/block", http.HandlerFunc(pprof.Handler("block").ServeHTTP))
	mux.Handle(prefix+"/cmdline", http.HandlerFunc(pprof.Cmdline))
	mux.Handle(prefix+"/goroutine", http.HandlerFunc(pprof.Handler("goroutine").ServeHTTP))
	mux.Handle(prefix+"/heap", http.HandlerFunc(pprof.Handler("heap").ServeHTTP))
	mux.Handle(prefix+"/mutex", http.HandlerFunc(pprof.Handler("mutex").ServeHTTP))
	mux.Handle(prefix+"/profile", http.HandlerFunc(pprof.Profile))
	mux.Handle(prefix+"/symbol", http.HandlerFunc(pprof.Symbol))
	mux.Handle(prefix+"/threadcreate", http.HandlerFunc(pprof.Handler("threadcreate").ServeHTTP))
	mux.Handle(prefix+"/trace", http.HandlerFunc(pprof.Trace))

	srv := http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		err := srv.ListenAndServe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "pprof: %v", err)
		}
	}()
}
