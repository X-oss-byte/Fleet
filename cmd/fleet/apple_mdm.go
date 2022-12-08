package main

import (
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/micromdm/nanomdm/certverify"
	httpmdm "github.com/micromdm/nanomdm/http/mdm"
	nanomdm_log "github.com/micromdm/nanomdm/log"
	nanomdm_service "github.com/micromdm/nanomdm/service"
	"github.com/micromdm/nanomdm/service/certauth"
	"github.com/micromdm/nanomdm/service/nanomdm"
	scep_depot "github.com/micromdm/scep/v2/depot"
	scepserver "github.com/micromdm/scep/v2/server"
)

// registerAppleMDMProtocolServices registers the HTTP handlers that serve
// the MDM services to Apple devices.
func registerAppleMDMProtocolServices(
	mux *http.ServeMux,
	scepConfig config.MDMAppleSCEPConfig,
	scepCertPEM []byte,
	scepKeyPEM []byte,
	mdmStorage *mysql.NanoMDMStorage,
	scepStorage *apple_mdm.SCEPMySQLDepot,
	logger kitlog.Logger,
) error {
	if err := registerSCEP(mux, scepConfig, scepCertPEM, scepKeyPEM, scepStorage, logger); err != nil {
		return fmt.Errorf("scep: %w", err)
	}
	if err := registerMDM(mux, scepCertPEM, mdmStorage, logger); err != nil {
		return fmt.Errorf("mdm: %w", err)
	}
	return nil
}

// registerSCEP registers the HTTP handler for SCEP service needed for enrollment to MDM.
// Returns the SCEP CA certificate that can be used by verifiers.
func registerSCEP(
	mux *http.ServeMux,
	scepConfig config.MDMAppleSCEPConfig,
	scepCertPEM []byte,
	scepKeyPEM []byte,
	scepStorage *apple_mdm.SCEPMySQLDepot,
	logger kitlog.Logger,
) error {
	scepCACert, err := apple_mdm.DecodeCertPEM(scepCertPEM)
	if err != nil {
		return fmt.Errorf("load SCEP CA certificate: %w", err)
	}

	scepCAKey, err := apple_mdm.DecodePrivateKeyPEM(scepKeyPEM)
	if err != nil {
		return fmt.Errorf("load SCEP CA private key: %w", err)
	}

	var signer scepserver.CSRSigner = scep_depot.NewSigner(
		scepStorage,
		scep_depot.WithValidityDays(scepConfig.Signer.ValidityDays),
		scep_depot.WithAllowRenewalDays(scepConfig.Signer.AllowRenewalDays),
	)
	scepChallenge := scepConfig.Challenge
	if scepChallenge == "" {
		return errors.New("missing SCEP challenge")
	}

	signer = scepserver.ChallengeMiddleware(scepChallenge, signer)
	scepService, err := scepserver.NewService(scepCACert, scepCAKey, signer,
		scepserver.WithLogger(kitlog.With(logger, "component", "mdm-apple-scep")),
	)
	if err != nil {
		return fmt.Errorf("initialize SCEP service: %w", err)
	}
	scepLogger := kitlog.With(logger, "component", "http-mdm-apple-scep")
	e := scepserver.MakeServerEndpoints(scepService)
	e.GetEndpoint = scepserver.EndpointLoggingMiddleware(scepLogger)(e.GetEndpoint)
	e.PostEndpoint = scepserver.EndpointLoggingMiddleware(scepLogger)(e.PostEndpoint)
	scepHandler := scepserver.MakeHTTPHandler(e, scepService, scepLogger)
	mux.Handle(apple_mdm.SCEPPath, scepHandler)
	return nil
}

// NanoMDMLogger is a logger adapter for nanomdm.
type NanoMDMLogger struct {
	logger kitlog.Logger
}

func NewNanoMDMLogger(logger kitlog.Logger) *NanoMDMLogger {
	return &NanoMDMLogger{
		logger: logger,
	}
}

func (l *NanoMDMLogger) Info(keyvals ...interface{}) {
	level.Info(l.logger).Log(keyvals...)
}

func (l *NanoMDMLogger) Debug(keyvals ...interface{}) {
	level.Debug(l.logger).Log(keyvals...)
}

func (l *NanoMDMLogger) With(keyvals ...interface{}) nanomdm_log.Logger {
	newLogger := kitlog.With(l.logger, keyvals...)
	return &NanoMDMLogger{
		logger: newLogger,
	}
}

// registerMDM registers the HTTP handlers that serve core MDM services (like checking in for MDM commands).
func registerMDM(
	mux *http.ServeMux,
	scepCAPEM []byte,
	mdmStorage *mysql.NanoMDMStorage,
	logger kitlog.Logger,
) error {
	certVerifier, err := certverify.NewPoolVerifier(scepCAPEM, x509.ExtKeyUsageClientAuth)
	if err != nil {
		return fmt.Errorf("certificate pool verifier: %w", err)
	}
	mdmLogger := NewNanoMDMLogger(kitlog.With(logger, "component", "http-mdm-apple-mdm"))

	// As usual, handlers are applied from bottom to top:
	// 1. Extract and verify MDM signature.
	// 2. Verify signer certificate with CA.
	// 3. Verify new or enrolled certificate (certauth.CertAuth which wraps the MDM service)
	// 4. Run actual MDM service operation (checkin handler or command and results handler).
	var mdmService nanomdm_service.CheckinAndCommandService = nanomdm.New(mdmStorage, nanomdm.WithLogger(mdmLogger))
	mdmService = certauth.New(mdmService, mdmStorage)
	var mdmHandler http.Handler = httpmdm.CheckinAndCommandHandler(mdmService, mdmLogger.With("handler", "checkin-command"))
	mdmHandler = httpmdm.CertVerifyMiddleware(mdmHandler, certVerifier, mdmLogger.With("handler", "cert-verify"))
	mdmHandler = httpmdm.CertExtractMdmSignatureMiddleware(mdmHandler, mdmLogger.With("handler", "cert-extract"))
	mux.Handle(apple_mdm.MDMPath, mdmHandler)
	return nil
}