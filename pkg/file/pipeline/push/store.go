// Copyright 2020 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package push

import (
	"context"
	"errors"

	"github.com/ethersphere/bee/pkg/file/pipeline"
	"github.com/ethersphere/bee/pkg/pushsync"
	"github.com/ethersphere/bee/pkg/storage"
	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/bee/pkg/topology"
	"golang.org/x/sync/errgroup"
)

var errInvalidData = errors.New("store: invalid data")

const concurrentPushes = 10

type pushWriter struct {
	ctx  context.Context
	next pipeline.ChainWriter
	ls   storage.Putter
	p    pushsync.PushSyncer

	sem chan struct{}
	eg  errgroup.Group
}

// NewPushSyncWriter returns a pushWriter. It writes the given data to the network
// using PushSyncer.
func NewPushSyncWriter(ctx context.Context, p pushsync.PushSyncer, next pipeline.ChainWriter) pipeline.ChainWriter {
	return &pushWriter{
		ctx:  ctx,
		p:    p,
		sem:  make(chan struct{}, concurrentPushes),
		next: next,
	}
}

func (w *pushWriter) ChainWrite(p *pipeline.PipeWriteArgs) error {
	if p.Ref == nil || p.Data == nil {
		return errInvalidData
	}
	var err error
	c := swarm.NewChunk(swarm.NewAddress(p.Ref), p.Data)
	/// TODO add stamp ^^
	w.sem <- struct{}{}
	w.eg.Go(func() error {
		defer func() {
			<-w.sem
		}()

	PUSH:
		select {
		case <-w.ctx.Done():
			return w.ctx.Err()
		default:
		}
		_, err = w.p.PushChunkToClosest(w.ctx, c)
		if err != nil {
			if errors.Is(err, topology.ErrWantSelf) {
				// if we are the closest, pushsync has already taken care
				// of replication, and we should put the chunk as ModePutSync
				_, err = w.ls.Put(w.ctx, storage.ModePutSync, c)
				if err != nil {

				}
			}
			// if the error is different, we keep on trying, hopefully
			// it gets through eventually
			goto PUSH
		}
		return nil
	})

	// this is here because the short pipeline used by the hashtrie writer
	// does not have a next writer to write to.
	if w.next == nil {
		// this assures that the hashtrie writer will not return the
		// hash to the caller before all intermediate chunks have been written
		// to the network.
		if err := w.eg.Wait(); err != nil {
			return err
		}
		return nil
	}

	return w.next.ChainWrite(p)
}

func (w *pushWriter) Sum() ([]byte, error) {
	err := w.eg.Wait()
	if err != nil {
		return nil, err
	}
	return w.next.Sum()
}
