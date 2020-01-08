package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/capacity"
	"github.com/threefoldtech/zos/pkg/capacity/dmi"
	"github.com/threefoldtech/zos/pkg/schema"

	"github.com/threefoldtech/zos/pkg/gedis/types/directory"
)

var prefixNode = []byte("node:")

type nodeStore struct {
	db *badger.DB
}

func NewNodeStore(db *badger.DB) *nodeStore {
	return &nodeStore{db: db}
}

func (s *nodeStore) Close() error {
	return nil
}

func nodeKey(nodeID string) []byte {
	return append(prefixNode, []byte(nodeID)...)
}

func (s *nodeStore) List() ([]*directory.TfgridNode2, error) {
	out := make([]*directory.TfgridNode2, 0, 100)

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(prefixNode); it.ValidForPrefix(prefixNode); it.Next() {
			item := it.Item()
			var node directory.TfgridNode2
			err := item.Value(func(v []byte) error {
				return json.Unmarshal(v, &node)
			})
			if err != nil {
				return err
			}
			out = append(out, &node)
		}
		return nil
	})

	return out, err
}

func (s *nodeStore) Get(nodeID string) (node directory.TfgridNode2, err error) {
	err = s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(nodeKey(nodeID))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &node)
		})
	})

	return node, err
}

func (s *nodeStore) Add(node directory.TfgridNode2) error {
	return s.db.Update(func(txn *badger.Txn) error {
		key := nodeKey(node.NodeID)

		item, err := txn.Get(key)

		if err == nil { // existing node
			var existing directory.TfgridNode2
			if err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &existing)
			}); err != nil {
				return err
			}

			// only update those fields when the node already exist
			existing.FarmID = node.FarmID
			existing.OsVersion = node.OsVersion
			existing.Location = node.Location
			existing.Updated = schema.Date{Time: time.Now()}
			node = existing
		}

		node.Created = schema.Date{Time: time.Now()}
		node.Updated = schema.Date{Time: time.Now()}
		val, err := json.Marshal(node)
		if err != nil {
			return err
		}
		log.Printf("store node %v", string(key))
		return txn.Set(key, val)
	})
}

func (s *nodeStore) updateTotalCapacity(nodeID string, cap directory.TfgridNodeResourceAmount1) error {
	return s.updateCapacity(nodeID, "total", cap)
}
func (s *nodeStore) updateReservedCapacity(nodeID string, cap directory.TfgridNodeResourceAmount1) error {
	return s.updateCapacity(nodeID, "reserved", cap)
}
func (s *nodeStore) updateUsedCapacity(nodeID string, cap directory.TfgridNodeResourceAmount1) error {
	return s.updateCapacity(nodeID, "used", cap)
}

func (s *nodeStore) updateCapacity(nodeID string, t string, cap directory.TfgridNodeResourceAmount1) error {
	return s.db.Update(func(txn *badger.Txn) error {
		key := nodeKey(nodeID)

		item, err := txn.Get(key)
		if err != nil {
			return err
		}

		var node directory.TfgridNode2
		if err := item.Value(func(val []byte) error {
			return json.Unmarshal(val, &node)
		}); err != nil {
			return err
		}

		switch t {
		case "total":
			node.TotalResources = cap
		case "reserved":
			node.ReservedResources = cap
		case "used":
			node.UsedResources = cap
		default:
			return fmt.Errorf("unsupported capacity type: %v", t)
		}

		val, err := json.Marshal(node)
		if err != nil {
			return err
		}
		return txn.Set(key, val)
	})
}

func (s *nodeStore) updateUptime(nodeID string, uptime int64) error {
	return s.db.Update(func(txn *badger.Txn) error {
		key := nodeKey(nodeID)

		item, err := txn.Get(key)
		if err != nil {
			return err
		}

		var node directory.TfgridNode2
		if err := item.Value(func(val []byte) error {
			return json.Unmarshal(val, &node)
		}); err != nil {
			return err
		}

		node.Uptime = uptime
		node.Updated = schema.Date{Time: time.Now()}
		val, err := json.Marshal(node)
		if err != nil {
			return err
		}
		return txn.Set(key, val)
	})
}

func (s *nodeStore) StoreProof(nodeID string, dmi dmi.DMI, disks capacity.Disks) error {
	return s.db.Update(func(txn *badger.Txn) error {
		key := nodeKey(nodeID)

		item, err := txn.Get(key)
		if err != nil {
			return err
		}

		var node directory.TfgridNode2
		if err := item.Value(func(val []byte) error {
			return json.Unmarshal(val, &node)
		}); err != nil {
			return err
		}

		proof := directory.TfgridNodeProof1{
			Created: schema.Date{Time: time.Now()},
		}

		proof.Hardware = map[string]interface{}{
			"sections": dmi.Sections,
			"tooling":  dmi.Tooling,
		}
		proof.HardwareHash, err = hashProof(proof.Hardware)
		if err != nil {
			return err
		}

		proof.Disks = map[string]interface{}{
			"aggregator":  disks.Aggregator,
			"environment": disks.Environment,
			"devices":     disks.Devices,
			"tool":        disks.Tool,
		}
		proof.DiskHash, err = hashProof(proof.Disks)
		if err != nil {
			return err
		}

		// don't save the proof if we already have one with the same
		// hash/content
		for _, p := range node.Proofs {
			if proof.Equal(p) {
				return nil
			}
		}

		node.Proofs = append(node.Proofs, proof)

		val, err := json.Marshal(node)
		if err != nil {
			return err
		}
		return txn.Set(key, val)
	})
}

func (s *nodeStore) SetInterfaces(nodeID string, ifaces []directory.TfgridNodeIface1) error {
	return s.db.Update(func(txn *badger.Txn) error {
		key := nodeKey(nodeID)

		item, err := txn.Get(key)
		if err != nil {
			return err
		}

		var node directory.TfgridNode2
		if err := item.Value(func(val []byte) error {
			return json.Unmarshal(val, &node)
		}); err != nil {
			return err
		}

		node.Ifaces = ifaces

		val, err := json.Marshal(node)
		if err != nil {
			return err
		}
		return txn.Set(key, val)
	})
}

func (s *nodeStore) SetPublicConfig(nodeID string, cfg directory.TfgridNodePublicIface1) error {
	return s.db.Update(func(txn *badger.Txn) error {
		key := nodeKey(nodeID)

		item, err := txn.Get(key)
		if err != nil {
			return err
		}

		var node directory.TfgridNode2
		if err := item.Value(func(val []byte) error {
			return json.Unmarshal(val, &node)
		}); err != nil {
			return err
		}

		if node.PublicConfig == nil {
			cfg.Version = 0
		} else {
			cfg.Version = node.PublicConfig.Version + 1
		}
		node.PublicConfig = &cfg

		val, err := json.Marshal(node)
		if err != nil {
			return err
		}
		return txn.Set(key, val)
	})
}

func (s *nodeStore) SetWGPorts(nodeID string, ports []uint) error {
	return s.db.Update(func(txn *badger.Txn) error {
		key := nodeKey(nodeID)

		item, err := txn.Get(key)
		if err != nil {
			return err
		}

		var node directory.TfgridNode2
		if err := item.Value(func(val []byte) error {
			return json.Unmarshal(val, &node)
		}); err != nil {
			return err
		}

		node.WGPorts = ports

		val, err := json.Marshal(node)
		if err != nil {
			return err
		}
		return txn.Set(key, val)
	})
}

func (s *nodeStore) Requires(key string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nodeID, ok := mux.Vars(r)[key]
		if !ok {
			// programming error, we should panic in this case
			panic("invalid node-id key")
		}

		_, err := s.Get(nodeID)
		if err != nil {
			// node not found
			httpError(w, errors.Wrapf(err, "node not found: %s", nodeID), http.StatusNotFound)
			return
		}

		handler(w, r)
	}
}

// hashProof return the hex encoded md5 hash of the json encoded version of p
func hashProof(p map[string]interface{}) (string, error) {

	// we are trying to have always produce same hash for same content of p
	// so we convert the map into a list so we can sort
	// the key and workaround the fact that maps are not sorted

	type kv struct {
		k string
		v interface{}
	}

	kvs := make([]kv, len(p))
	for k, v := range p {
		kvs = append(kvs, kv{k: k, v: v})
	}
	sort.Slice(kvs, func(i, j int) bool { return kvs[i].k < kvs[j].k })

	b, err := json.Marshal(kvs)
	if err != nil {
		return "", err
	}
	h := md5.New()
	bh := h.Sum(b)
	return fmt.Sprintf("%x", bh), nil
}
