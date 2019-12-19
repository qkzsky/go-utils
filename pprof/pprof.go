package pprof

import (
	"log"
	"net/http"
	"net/http/pprof"
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
	mux.Handle(prefix+"/allocs", pprof.Handler("allocs"))
	mux.Handle(prefix+"/block", pprof.Handler("block"))
	mux.Handle(prefix+"/cmdline", http.HandlerFunc(pprof.Cmdline))
	mux.Handle(prefix+"/goroutine", pprof.Handler("goroutine"))
	mux.Handle(prefix+"/heap", pprof.Handler("heap"))
	mux.Handle(prefix+"/mutex", pprof.Handler("mutex"))
	mux.Handle(prefix+"/profile", http.HandlerFunc(pprof.Profile))
	mux.Handle(prefix+"/symbol", http.HandlerFunc(pprof.Symbol))
	mux.Handle(prefix+"/threadcreate", pprof.Handler("threadcreate"))
	mux.Handle(prefix+"/trace", http.HandlerFunc(pprof.Trace))

	srv := http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		err := srv.ListenAndServe()
		if err != nil {
			log.Println("[pprof] " + err.Error())
		}
	}()
}
