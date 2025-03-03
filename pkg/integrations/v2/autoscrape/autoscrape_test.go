package autoscrape

import (
	"context"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafana/agent/pkg/metrics/instance"
	"github.com/grafana/agent/pkg/util"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/model"
	prom_config "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/pkg/exemplar"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAutoscrape is a basic end-to-end test of the autoscraper.
func TestAutoscrape(t *testing.T) {
	srv := httptest.NewServer(promhttp.Handler())
	defer srv.Close()

	wt := util.NewWaitTrigger()

	noop := noOpAppender
	noop.AppendFunc = func(ref uint64, l labels.Labels, t int64, v float64) (uint64, error) {
		wt.Trigger()
		return noOpAppender.AppendFunc(ref, l, t, v)
	}

	im := instance.MockManager{
		GetInstanceFunc: func(name string) (instance.ManagedInstance, error) {
			assert.Equal(t, t.Name(), name)
			return &mockInstance{app: &noop}, nil
		},
	}
	as := NewScraper(util.TestLogger(t), im)
	defer as.Stop()

	err := as.ApplyConfig([]*ScrapeConfig{{
		Instance: t.Name(),
		Config: func() prom_config.ScrapeConfig {
			cfg := prom_config.DefaultScrapeConfig
			cfg.JobName = t.Name()
			cfg.ScrapeInterval = model.Duration(time.Second)
			cfg.ServiceDiscoveryConfigs = discovery.Configs{
				discovery.StaticConfig{{
					Targets: []model.LabelSet{{
						model.AddressLabel: model.LabelValue(srv.Listener.Addr().String()),
					}},
					Source: t.Name(),
				}},
			}
			return cfg
		}(),
	}})
	require.NoError(t, err, "failed to apply configs")

	// NOTE(rfratto): SD won't start sending targets until after 5 seconds. We'll
	// need to at least wait that long.
	time.Sleep(5 * time.Second)

	require.NoError(t, wt.Wait(5*time.Second), "timed out waiting for scrape")
}

var globalRef uint64
var noOpAppender = mockAppender{
	AppendFunc: func(ref uint64, l labels.Labels, t int64, v float64) (uint64, error) {
		return atomic.AddUint64(&globalRef, 1), nil
	},
	CommitFunc:   func() error { return nil },
	RollbackFunc: func() error { return nil },
	AppendExemplarFunc: func(ref uint64, l labels.Labels, e exemplar.Exemplar) (uint64, error) {
		return atomic.AddUint64(&globalRef, 1), nil
	},
}

type mockAppender struct {
	AppendFunc         func(ref uint64, l labels.Labels, t int64, v float64) (uint64, error)
	CommitFunc         func() error
	RollbackFunc       func() error
	AppendExemplarFunc func(ref uint64, l labels.Labels, e exemplar.Exemplar) (uint64, error)
}

func (ma *mockAppender) Append(ref uint64, l labels.Labels, t int64, v float64) (uint64, error) {
	return ma.AppendFunc(ref, l, t, v)
}
func (ma *mockAppender) Commit() error   { return ma.CommitFunc() }
func (ma *mockAppender) Rollback() error { return ma.RollbackFunc() }
func (ma *mockAppender) AppendExemplar(ref uint64, l labels.Labels, e exemplar.Exemplar) (uint64, error) {
	return ma.AppendExemplarFunc(ref, l, e)
}

type mockInstance struct {
	instance.NoOpInstance
	app storage.Appender
}

func (mi *mockInstance) Appender(ctx context.Context) storage.Appender { return mi.app }
