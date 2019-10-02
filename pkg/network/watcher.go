package network

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zos/pkg"
)

// Watcher is an object that is responsible to
// watch the tnodb for update networks object
type Watcher struct {
	nodeID pkg.Identifier
	db     TNoDBUtils
}

// NewWatcher creates a new watcher for a specific node
func NewWatcher(nodeID pkg.Identifier, db TNoDBUtils) *Watcher {
	return &Watcher{
		nodeID: nodeID,
		db:     db,
	}
}

// Watch starts a gorountine that will poll the tnodb for new version
// of a network object
// it returns a channel of network ID that have a new version
func (w *Watcher) Watch(ctx context.Context) <-chan pkg.NetID {
	versions := make(map[pkg.NetID]uint32)

	ch := make(chan pkg.NetID)
	go func() {
		defer close(ch)

		for {
			newVersions, err := w.db.GetNetworksVersion(w.nodeID)
			if err != nil {
				log.Error().
					Err(err).Msg("fail to get network versions")
				continue
			}

			toSend := []pkg.NetID{}

			for netID, newVersion := range newVersions {
				v, ok := versions[netID]
				if !ok {
					toSend = append(toSend, netID)
				} else if newVersion > v {
					toSend = append(toSend, netID)
				}
				versions[netID] = newVersion

				select {
				case <-ctx.Done():
					break
				default:
					for _, netID := range toSend {
						ch <- netID
					}
				}
			}

			select {
			case <-ctx.Done():
				break
			case <-time.After(time.Second * 20):
			}
		}
	}()

	return ch
}
