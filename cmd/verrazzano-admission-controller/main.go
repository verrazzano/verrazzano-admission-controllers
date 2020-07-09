// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/rs/zerolog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/verrazzano/verrazzano-admission-controllers/pkg"
)

const (
	port = "8080"
)

var (
	tlscert string
	tlskey  string
)

func main() {

	flag.StringVar(&tlscert, "tlsCertFile", "/etc/certs/cert.pem", "File containing the x509 Certificate for HTTPS.")
	flag.StringVar(&tlskey, "tlsKeyFile", "/etc/certs/key.pem", "File containing the x509 private key to --tlsCertFile.")

	flag.Parse()
	InitLogs()

	// create initial logger with predefined elements
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "AdmissionController").Str("name", "AdmissionInit").Logger()

	logger.Info().Msg("Starting Verrazzano validation admission controller")

	certs, err := tls.LoadX509KeyPair(tlscert, tlskey)
	if err != nil {
		logger.Error().Msgf("Failed to load key pair: %v", err)
	}

	// define http server and server handler
	server := &http.Server{
		Addr:      fmt.Sprintf(":%v", port),
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{certs}},
	}
	sh := pkg.ServerHandler{}
	mux := http.NewServeMux()
	mux.HandleFunc("/validate", sh.Serve)
	server.Handler = mux

	// start webhook server
	go func() {
		if err := server.ListenAndServeTLS("", ""); err != nil {
			logger.Error().Msgf("Failed to listen and serve webhook server: %v", err)
		}
	}()

	logger.Info().Msgf("Server running listening in port: %s", port)

	// listen for shutdown signal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	logger.Info().Msg("Got shutdown signal, shutting down webhook server gracefully...")
	server.Shutdown(context.Background())
}
