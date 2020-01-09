package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"

	"github.com/threefoldtech/zos/pkg/gedis/types/directory"
)

func (s *farmStore) registerFarm(w http.ResponseWriter, r *http.Request) {
	log.Println("farm register request received")

	defer r.Body.Close()

	info := directory.TfgridFarm1{}
	if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := s.Add(info)
	if err != nil {
		httpError(w, err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(struct {
		ID pkg.FarmID `json:"id"`
	}{
		id,
	})
}

func (s *farmStore) listFarm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	farms, err := s.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(farms)
}

func (s *farmStore) cockpitListFarm(w http.ResponseWriter, r *http.Request) {
	farms, err := s.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	x := struct {
		Farms []*directory.TfgridFarm1 `json:"farms"`
	}{farms}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(x)
}

func (s *farmStore) getFarm(w http.ResponseWriter, r *http.Request) {
	sid := mux.Vars(r)["farm_id"]

	id, err := strconv.ParseUint(sid, 10, 64)
	if err != nil {
		httpError(w, errors.Wrap(err, "id should be an integer"), http.StatusBadRequest)
		return
	}

	farm, err := s.GetByID(id)
	if err != nil {
		httpError(w, err, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(farm)
}
