package main

import (
	"sync"
	"time"

	"github.com/DataDog/raclette/config"
	"github.com/DataDog/raclette/model"
	"github.com/DataDog/raclette/quantizer"
	log "github.com/cihub/seelog"
)

type processedTrace struct {
	Trace     model.Trace
	Root      *model.Span
	Env       string
	Sublayers []model.SublayerValue
}

// Agent struct holds all the sub-routines structs and make the data flow between them
type Agent struct {
	Receiver     *HTTPReceiver
	Concentrator *Concentrator
	Sampler      *Sampler
	Writer       *Writer

	// config
	conf *config.AgentConfig

	// Used to synchronize on a clean exit
	exit chan struct{}
}

// NewAgent returns a new Agent object, ready to be started
func NewAgent(conf *config.AgentConfig) *Agent {
	exit := make(chan struct{})

	r := NewHTTPReceiver(conf)
	c := NewConcentrator(
		conf.ExtraAggregators,
		conf.BucketInterval.Nanoseconds(),
	)
	s := NewSampler(conf)

	w := NewWriter(conf)
	w.inServices = r.services

	return &Agent{
		Receiver:     r,
		Concentrator: c,
		Sampler:      s,
		Writer:       w,
		conf:         conf,
		exit:         exit,
	}
}

// Run starts routers routines and individual pieces then stop them when the exit order is received
func (a *Agent) Run() {
	flushTicker := time.NewTicker(a.conf.BucketInterval)
	defer flushTicker.Stop()

	a.Receiver.Run()
	go a.Writer.Run()

	for {
		select {
		case t := <-a.Receiver.traces:
			a.Process(t)
		case <-flushTicker.C:
			p := model.AgentPayload{
				HostName: a.conf.HostName,
				Env:      a.conf.DefaultEnv,
			}
			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				defer wg.Done()
				p.Stats = a.Concentrator.Flush()
			}()
			go func() {
				defer wg.Done()
				p.Traces = a.Sampler.Flush()
			}()

			a.Writer.inPayloads <- p
		case <-a.exit:
			break
		}
	}

	log.Info("exiting")
	close(a.Receiver.exit)
	close(a.Writer.exit)
}

func (a *Agent) Process(t model.Trace) {
	if len(t) == 0 {
		// TODO: empty trace ++ / debug log
		return
	}

	sublayers := model.ComputeSublayers(&t)
	root := t.GetRoot()
	model.PinSublayersOnSpan(root, sublayers)

	if root.End() < model.Now()-a.conf.OldestSpanCutoff {
		// TODO: late trace ++ / debug log
		return
	}

	for i := range t {
		t[i] = quantizer.Quantize(t[i])
	}

	pt := processedTrace{
		Trace:     t,
		Root:      root,
		Env:       t.GetEnv(),
		Sublayers: sublayers,
	}

	// NOTE: right now we don't use the .Metrics map in the concentrator
	// but if we did, it would be racy with the Sampler that edits it
	go a.Concentrator.Add(pt)
	go a.Sampler.Add(pt)
}
