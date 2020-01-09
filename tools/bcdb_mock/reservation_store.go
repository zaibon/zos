package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/dgraph-io/badger"
	"github.com/threefoldtech/zos/pkg/provision"
)

type reservation struct {
	Reservation *provision.Reservation `json:"reservation"`
	Result      *provision.Result      `json:"result"`
	Deleted     bool                   `json:"deleted"`
	NodeID      string                 `json:"node_id"`
}

var (
	prefixReservation = []byte("reservation:")
)

type reservationsStore struct {
	db  *badger.DB
	seq *badger.Sequence
}

func NewReservationStore(db *badger.DB) *reservationsStore {
	seq, err := db.GetSequence([]byte("reservation_seq"), 10)
	if err != nil {
		log.Fatalf("error creating farm sequence: %v", err)
	}
	return &reservationsStore{
		db:  db,
		seq: seq,
	}
}
func (s *reservationsStore) Close() error {
	return s.seq.Release()
}

func (s *reservationsStore) List() ([]*reservation, error) {
	out := make([]*reservation, 0, 100)

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(prefixReservation); it.ValidForPrefix(prefixReservation); it.Next() {
			item := it.Item()
			var res reservation
			err := item.Value(func(v []byte) error {
				return json.Unmarshal(v, &res)
			})
			if err != nil {
				return err
			}
			out = append(out, &res)
		}
		return nil
	})

	return out, err
}

func (s *reservationsStore) Get(ID string) (res *reservation, err error) {
	err = s.db.View(func(txn *badger.Txn) error {
		key := append(prefixReservation, []byte(ID)...)
		item, err := txn.Get(key)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &res)
		})
	})

	return res, err
}

func (s *reservationsStore) Add(nodeID string, res *provision.Reservation) error {
	return s.db.Update(func(txn *badger.Txn) error {
		total, err := s.seq.Next()
		if err != nil {
			return err
		}

		res.ID = fmt.Sprintf("%d-1", total)
		key := append(prefixReservation, []byte(res.ID)...)

		val, err := json.Marshal(&reservation{
			NodeID:      nodeID,
			Reservation: res,
		})
		if err != nil {
			return err
		}

		return txn.Set(key, val)
	})
}

func (s *reservationsStore) GetReservations(nodeID string, from uint64) ([]*provision.Reservation, error) {
	output := []*provision.Reservation{}
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(prefixReservation); it.ValidForPrefix(prefixReservation); it.Next() {
			item := it.Item()

			var r reservation
			err := item.Value(func(v []byte) error {
				return json.Unmarshal(v, &r)
			})
			if err != nil {
				return err
			}

			if r.NodeID != nodeID {
				continue
			}

			resID, _, err := r.Reservation.SplitID()
			if err != nil {
				continue
			}

			if from == 0 ||
				(!r.Reservation.Expired() && resID >= from) ||
				(r.Reservation.ToDelete && !r.Deleted) {
				output = append(output, r.Reservation)
			}
		}
		return nil
	})
	return output, err
}
