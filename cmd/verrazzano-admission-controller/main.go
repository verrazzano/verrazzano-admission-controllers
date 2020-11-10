// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/verrazzano/verrazzano-admission-controllers/pkg"
	"go.uber.org/zap"
	kzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	port = "8080"
)

var (
	tlscert       string
	tlskey        string
	verrazzanoURI string
	zapOptions    = kzap.Options{}
)

func main() {
	flag.StringVar(&tlscert, "tlsCertFile", "/etc/certs/cert.pem", "File containing the x509 Certificate for HTTPS.")
	flag.StringVar(&tlskey, "tlsKeyFile", "/etc/certs/key.pem", "File containing the x509 private key to --tlsCertFile.")
	flag.StringVar(&verrazzanoURI, "verrazzanoUri", "", "Verrazzano URI, for example my-verrazzano-1.verrazzano.example.com")
	zapOptions.BindFlags(flag.CommandLine)
	flag.Parse()
	InitLogs(zapOptions)

	zap.S().Infof("Starting Verrazzano validation admission controller")

	certs, err := tls.LoadX509KeyPair(tlscert, tlskey)
	if err != nil {
		zap.S().Errorf("Failed to load key pair: %v", err)
	}

	// define http server and server handler
	server := &http.Server{
		Addr:      fmt.Sprintf(":%v", port),
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{certs}},
	}
	sh := pkg.ServerHandler{
		VerrazzanoURI: verrazzanoURI,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/validate", sh.Serve)
	server.Handler = mux

	// start webhook server
	go func() {
		if err := server.ListenAndServeTLS("", ""); err != nil {
			zap.S().Errorf("Failed to listen and serve webhook server: %v", err)
		}
	}()

	zap.S().Infof("Server running listening in port: %s", port)

	// listen for shutdown signal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	zap.S().Infow("Got shutdown signal, shutting down webhook server gracefully...")
	server.Shutdown(context.Background())
}
