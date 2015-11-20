package main

import (
	log "github.com/cihub/seelog"

	"github.com/DataDog/raclette/config"
	"github.com/DataDog/raclette/model"
	"github.com/DataDog/raclette/sampler"
)

// Sampler chooses wich spans to write to the API
type Sampler struct {
	inSpans    chan model.Span
	inPayloads chan model.AgentPayload // Trigger the flush of the sampler when stats are received
	out        chan model.AgentPayload // Output the stats + samples

	conf *config.AgentConfig

	se SamplerEngine

	Worker
}

// SamplerEngine cares about ingesting spans and stats to return a sampled payload
type SamplerEngine interface {
	AddSpan(span model.Span)
	FlushPayload(sb model.AgentPayload) model.AgentPayload
}

// NewSampler creates a new empty sampler
func NewSampler(
	inSpans chan model.Span, inPayloads chan model.AgentPayload, conf *config.AgentConfig,
) *Sampler {
	s := &Sampler{
		inSpans:    inSpans,
		inPayloads: inPayloads,
		out:        make(chan model.AgentPayload),

		conf: conf,

		se: sampler.NewResourceQuantileSampler(conf),
	}
	s.Init()
	return s
}

// Start runs the writer by consuming spans in a buffer and periodically
// flushing to the API
func (s *Sampler) Start() {
	s.wg.Add(1)
	go s.run()

	log.Info("Sampler started")
}

// We rely on the concentrator ticker to flush periodically traces "aligning" on the buckets
// (it's not perfect, but we don't really care, traces of this stats bucket may arrive in the next flush)
func (s *Sampler) run() {
	for {
		select {
		case span := <-s.inSpans:
			s.se.AddSpan(span)
		case ap := <-s.inPayloads:
			log.Info("Received a payload, initiating a sampling + flush")
			s.out <- s.se.FlushPayload(ap)
		case <-s.exit:
			log.Info("Sampler exiting")
			s.wg.Done()
			return
		}
	}
}
