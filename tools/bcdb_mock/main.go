package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/dgraph-io/badger"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

var listen string
var storage string

func main() {
	flag.StringVar(&listen, "listen", ":8080", "listen address, default :8080")
	flag.StringVar(&storage, "data", "data", "path to the directory where to store the data")
	flag.Parse()

	db, err := badger.Open(badger.DefaultOptions(storage))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	nodeStore := NewNodeStore(db)
	farmStore := NewFarmStore(db)
	resStore := NewReservationStore(db)

	defer func() {
		if err := nodeStore.Close(); err != nil {
			log.Printf("error closing node store: %v", err)
		}
		if err := farmStore.Close(); err != nil {
			log.Printf("error closing farm store: %v", err)
		}
		if err := resStore.Close(); err != nil {
			log.Printf("error closing node store: %v", err)
		}
	}()

	router := mux.NewRouter()

	router.HandleFunc("/nodes", nodeStore.registerNode).Methods("POST")

	router.HandleFunc("/nodes/{node_id}", nodeStore.nodeDetail).Methods("GET")
	router.HandleFunc("/nodes/{node_id}/interfaces", nodeStore.registerIfaces).Methods("POST")
	router.HandleFunc("/nodes/{node_id}/ports", nodeStore.registerPorts).Methods("POST")
	router.HandleFunc("/nodes/{node_id}/configure_public", nodeStore.configurePublic).Methods("POST")
	router.HandleFunc("/nodes/{node_id}/capacity", nodeStore.registerCapacity).Methods("POST")
	router.HandleFunc("/nodes/{node_id}/uptime", nodeStore.updateUptimeHandler).Methods("POST")
	router.HandleFunc("/nodes", nodeStore.listNodes).Methods("GET")

	router.HandleFunc("/farms", farmStore.registerFarm).Methods("POST")
	router.HandleFunc("/farms", farmStore.listFarm).Methods("GET")
	router.HandleFunc("/farms/{farm_id}", farmStore.getFarm).Methods("GET")

	// compatibility with gedis_http
	router.HandleFunc("/nodes/list", nodeStore.cockpitListNodes).Methods("POST")
	router.HandleFunc("/farms/list", farmStore.cockpitListFarm).Methods("POST")

	router.HandleFunc("/reservations/{node_id}", nodeStore.Requires("node_id", resStore.reserve)).Methods("POST")
	router.HandleFunc("/reservations/{node_id}/poll", nodeStore.Requires("node_id", resStore.poll)).Methods("GET")
	router.HandleFunc("/reservations/{id}", resStore.get).Methods("GET")
	router.HandleFunc("/reservations/{id}", resStore.putResult).Methods("PUT")
	router.HandleFunc("/reservations/{id}/deleted", resStore.putDeleted).Methods("PUT")
	router.HandleFunc("/reservations/{id}", resStore.delete).Methods("DELETE")

	log.Printf("start on %s\n", listen)
	r := handlers.LoggingHandler(os.Stderr, router)
	r = handlers.CORS()(r)

	s := &http.Server{
		Addr:    listen,
		Handler: r,
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)

	go s.ListenAndServe()

	<-c

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		log.Printf("error during server shutdown: %v\n", err)
	}
}

// if importPath != "" {
// 	f, err := os.OpenFile(importPath, os.O_RDONLY, 0660)
// 	if err != nil {
// 		log.Fatalf("error during import: %v", err)
// 	}
// 	defer f.Close()

// 	nodes := struct {
// 		Nodes []*directory.TfgridNode2 `json:"nodes"`
// 	}{
// 		Nodes: []*directory.TfgridNode2{},
// 	}
// 	if err := json.NewDecoder(f).Decode(&nodes); err != nil {
// 		log.Fatal(err)
// 	}

// 	nodeStore := &nodeStore{db: db}
// 	for _, node := range nodes.Nodes {
// 		if err := nodeStore.Add(*node); err != nil {
// 			log.Fatalf("error during import: %v", err)
// 		}
// 	}

// 	return
// }
