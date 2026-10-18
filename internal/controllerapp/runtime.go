// SPDX-License-Identifier: Apache-2.0
package controllerapp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/PicketX/PicketX/internal/agentsync"
	"github.com/PicketX/PicketX/internal/controller"
	"github.com/PicketX/PicketX/internal/management"
	"github.com/PicketX/PicketX/internal/managementaccess"
	"github.com/PicketX/PicketX/internal/publication"
	"github.com/PicketX/PicketX/internal/runtimeconfig"
	"github.com/PicketX/PicketX/pkg/configuration"
	agentv1 "github.com/PicketX/PicketX/pkg/protocol/agent/v1alpha1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"log/slog"
	"net"
	"net/http"
	"time"
)

type AgentService interface {
	agentv1.AgentSyncServer
	ReplaceCredentials([]agentsync.Credential) error
	CheckPublication(*configuration.Candidate, uint64) error
	Publish(*configuration.Candidate, uint64) error
	Close()
}
type Dependencies struct {
	Listen     func(string, string) (net.Listener, error)
	NewService func(string, agentsync.Limits) (AgentService, error)
}

// Run owns the complete process lifecycle; publishing stops before service/DB close.
func Run(ctx context.Context, c runtimeconfig.Config, log *slog.Logger, d Dependencies) error {
	if e := c.Validate(); e != nil {
		return e
	}
	if log == nil {
		return fmt.Errorf("logger required")
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	var grpcOptions = []grpc.ServerOption{grpc.MaxRecvMsgSize(agentsync.MaxWireBytes), grpc.MaxSendMsgSize(agentsync.MaxWireBytes)}
	if c.Agent.TLSCert != "" {
		pair, e := tls.LoadX509KeyPair(c.Agent.TLSCert, c.Agent.TLSKey)
		if e != nil {
			return fmt.Errorf("load Agent TLS keypair: %w", e)
		}
		grpcOptions = append(grpcOptions, grpc.Creds(credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{pair}})))
	}
	if d.Listen == nil {
		d.Listen = net.Listen
	}
	if d.NewService == nil {
		d.NewService = func(a string, l agentsync.Limits) (AgentService, error) { return agentsync.New(a, l) }
	}
	httpListener, e := d.Listen("tcp", c.HTTPListen)
	if e != nil {
		return e
	}
	defer httpListener.Close()
	var agentListener net.Listener
	var agentServer *grpc.Server
	var p *publisher
	var access *managementaccess.Access
	var managementService *management.Service
	failures := make(chan error, 3)
	if c.Agent.Enabled {
		agentListener, e = d.Listen("tcp", c.Agent.Listen)
		if e != nil {
			return e
		}
		defer agentListener.Close()
		svc, e := d.NewService(c.Agent.Authority, c.Agent.Limits)
		if e != nil {
			return e
		}
		if svc == nil {
			return fmt.Errorf("Agent service factory returned nil")
		}
		defer svc.Close()
		p = &publisher{cfg: c, service: svc, log: log}
		p.fatal = func(e error) {
			select {
			case failures <- e:
			default:
			}
		}
		if !p.credentials() {
			return fmt.Errorf("initial Agent credentials rejected")
		}
		p.store, e = publication.OpenMode(c.Database, c.Agent.Authority, c.Source.Mode)
		if e != nil {
			return e
		}
		defer p.store.Close()
		if c.Source.Mode == "database" {
			access, e = managementaccess.New(p.store.GORM(), managementaccess.Settings{Credentials: c.Management.Credentials, Reload: c.Management.Reload, WarnRules: c.Management.WarnRules, WarnLoadPercent: c.Management.WarnLoadPercent}, log)
			if e != nil {
				return e
			}
			if e = p.initializeManaged(); e != nil {
				return e
			}
			managementService = management.New(access, p)
		}
		if candidate, r := p.store.Current(); candidate != nil {
			if e = svc.Publish(candidate, r); e != nil {
				return e
			}
		}
		if e = p.reload(); e != nil {
			return e
		}
		if candidate, _ := p.store.Current(); candidate == nil {
			return fmt.Errorf("no valid initial publication")
		}
		agentServer = grpc.NewServer(grpcOptions...)
		agentv1.RegisterAgentSyncServer(agentServer, svc)
	}
	ready := func() bool {
		if p == nil {
			return true
		}
		return p.ready.Load() && (access == nil || access.Ready())
	}
	server := &http.Server{Handler: controller.NewHandlerWithManagement(ready, managementService, controller.ManagementLimits{RequestsPerSecond: c.Management.RequestsPerSecond, Burst: c.Management.Burst}), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	accessDone := make(chan struct{})
	if access != nil {
		go func() { defer close(accessDone); access.Run(workerCtx) }()
	} else {
		close(accessDone)
	}
	pollDone := make(chan struct{})
	if p != nil {
		go func() {
			defer close(pollDone)
			ticker := time.NewTicker(c.Poll)
			defer ticker.Stop()
			for {
				select {
				case <-workerCtx.Done():
					return
				case <-ticker.C:
					if e := p.reload(); e != nil {
						failures <- e
						return
					}
				}
			}
		}()
	} else {
		close(pollDone)
	}
	go func() {
		e := server.Serve(httpListener)
		if e != nil && !errors.Is(e, http.ErrServerClosed) {
			failures <- e
		}
	}()
	agentAddress := ""
	if agentServer != nil {
		agentAddress = agentListener.Addr().String()
		go func() {
			e := agentServer.Serve(agentListener)
			if e != nil && !errors.Is(e, grpc.ErrServerStopped) {
				failures <- e
			}
		}()
	}
	log.Info("controller started", "http_listen", httpListener.Addr().String(), "agent_listen", agentAddress, "agent_enabled", c.Agent.Enabled)
	var runErr error
	select {
	case <-ctx.Done():
	case runErr = <-failures:
	}
	cancel()
	<-pollDone
	<-accessDone
	if p != nil {
		p.ready.Store(false)
		p.service.Close()
	}
	shutdownCtx, stop := context.WithTimeout(context.Background(), c.Shutdown)
	defer stop()
	httpDone := make(chan error, 1)
	go func() { httpDone <- server.Shutdown(shutdownCtx) }()
	if agentServer != nil {
		done := make(chan struct{})
		go func() { agentServer.GracefulStop(); close(done) }()
		select {
		case <-done:
		case <-shutdownCtx.Done():
			agentServer.Stop()
			<-done
		}
	}
	if e := <-httpDone; e != nil {
		_ = server.Close()
		if !errors.Is(e, context.DeadlineExceeded) {
			runErr = errors.Join(runErr, e)
		}
	}
	return runErr
}
