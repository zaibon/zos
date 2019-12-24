package main

import (
	"context"
	"fmt"

	"github.com/gizak/termui/v3/widgets"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/stubs"
)

func headerRenderer(c zbus.Client, h *widgets.Paragraph, r func()) {
	env, _ := environment.Get()
	format := fmt.Sprintf("Zero OS [%s] Version: %%s", env.RunningMode.String())

	h.Text = "Zero OS"

	host := stubs.NewHostMonitorStub(c)
	ctx := context.Background()
	ch, err := host.Version(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to start update stream for version")
	}

	go func() {
		for version := range ch {
			v := fmt.Sprintf(format, version.String())
			if h.Text != v {
				h.Text = v
				r()
			}
		}
	}()
}
