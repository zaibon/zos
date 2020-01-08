package main

import (
	"encoding/binary"
	"encoding/json"
	"log"

	"github.com/dgraph-io/badger"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gedis/types/directory"
)

var (
	prefixFarm = []byte("farm:")
)

type farmStore struct {
	db  *badger.DB
	seq *badger.Sequence
}

func NewFarmStore(db *badger.DB) *farmStore {
	seq, err := db.GetSequence([]byte("farm_seq"), 10)
	if err != nil {
		log.Fatalf("error creating farm sequence: %v", err)
	}
	return &farmStore{
		db:  db,
		seq: seq,
	}
}
func (s *farmStore) Close() error {
	return s.seq.Release()
}

func farmKey(id uint64) []byte {
	bs := make([]byte, 8)
	binary.LittleEndian.PutUint64(bs, id)
	return append(prefixFarm, bs...)
}

func farmID(b []byte) uint64 {
	return binary.LittleEndian.Uint64(b[len(prefixFarm):])
}

func (s *farmStore) List() ([]*directory.TfgridFarm1, error) {
	out := make([]*directory.TfgridFarm1, 0, 100)

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(prefixFarm); it.ValidForPrefix(prefixFarm); it.Next() {
			item := it.Item()
			var farm directory.TfgridFarm1
			err := item.Value(func(v []byte) error {
				return json.Unmarshal(v, &farm)
			})
			if err != nil {
				return err
			}
			out = append(out, &farm)
		}
		return nil
	})

	return out, err
}

func (s *farmStore) GetByID(id uint64) (farm *directory.TfgridFarm1, err error) {
	err = s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(farmKey(id))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &farm)
		})
	})

	return farm, err
}

func (s *farmStore) Add(farm directory.TfgridFarm1) (pkg.FarmID, error) {
	err := s.db.Update(func(txn *badger.Txn) error {

		item, err := txn.Get([]byte(farm.Name))
		if err == nil { // existing farm
			var existing directory.TfgridFarm1
			if err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &existing)
			}); err != nil {
				return err
			}

			// only update those fields when the farm already exist
			existing.WalletAddresses = farm.WalletAddresses
			existing.Location = farm.Location
			existing.Email = farm.Email
			existing.ResourcePrices = farm.ResourcePrices

			farm = existing
		}

		total, err := s.seq.Next()
		if err != nil {
			return err
		}
		farm.ID = uint64(total + 1) // ids starts at 1
		key := farmKey(farm.ID)

		val, err := json.Marshal(farm)
		if err != nil {
			return err
		}

		if err := txn.Set(key, val); err != nil {
			return err
		}
		return txn.Set([]byte(farm.Name), val)
	})

	return pkg.FarmID(farm.ID), err
}
