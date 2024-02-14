package cmdutil

import (
	"fmt"
	"net/http"
	"net/http/pprof"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

type pprofConfig struct {
	listenAddr string
}

var pprofCfg = pprofConfig{
	listenAddr: "127.0.0.1:3031",
}

func RegisterPprofFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(
		&pprofCfg.listenAddr,
		"pprof-listen-addr",
		pprofCfg.listenAddr,
		"Address for the pprof endpoint to listen to.",
	)
	// Hiding the flag since this should be for
	// internal use. But we can document later on
	// that its on port 3031

	//nolint:errcheck
	cmd.Flags().MarkHidden("pprof-listen-addr")
}

func PprofServer(logger zerolog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := fmt.Fprint(w, "OK"); err != nil {
			logger.Err(err).Msgf("error writing to healthz")
		}
	})
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	return mux
}

func RunPprofServer(logger zerolog.Logger) {
	go func() {
		m := PprofServer(logger)
		if err := http.ListenAndServe(pprofCfg.listenAddr, m); err != nil {
			logger.Err(err).Msgf("error exposing pprof endpoints")
		}
	}()
}
