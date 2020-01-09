package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/threefoldtech/zos/pkg/gedis/types/directory"
	"github.com/urfave/cli"

	"github.com/dgraph-io/badger"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func main() {
	app := cli.NewApp()
	app.Usage = "BCDB mock"
	app.Version = "0.0.1"
	app.EnableBashCompletion = true

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "listen, l",
			Usage: "listen address, default :8080",
			Value: ":8080",
		},
		cli.StringFlag{
			Name:  "data, d",
			Usage: "path to the directory where to store the data",
		},
	}
	app.Action = func(c *cli.Context) error {
		storage := c.String("data")
		listen := c.String("listen")

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

		cExit := make(chan os.Signal)
		signal.Notify(cExit, os.Interrupt)

		go s.ListenAndServe()

		<-cExit

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		if err := s.Shutdown(ctx); err != nil {
			log.Printf("error during server shutdown: %v\n", err)
			return err
		}
		return nil
	}

	app.Commands = []cli.Command{
		cli.Command{
			Name: "import",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "data, d",
					Usage: "path to the directory where to store the data",
				},
				cli.StringFlag{
					Name:  "path, p",
					Usage: "path to the data file to import",
				},
				cli.StringFlag{
					Name:  "type",
					Usage: "value of data to import. Can be 'node', 'farm' or 'reservation'",
				},
			},
			Action: func(c *cli.Context) error {
				d := c.String("data")
				p := c.String("path")
				t := c.String("type")

				db, err := badger.Open(badger.DefaultOptions(d))
				if err != nil {
					log.Fatal(err)
				}
				defer db.Close()

				return importData(p, t, db)
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func importData(p, t string, db *badger.DB) error {
	if p == "" {
		return fmt.Errorf("path cannot be empty")
	}

	f, err := os.Open(p)
	if err != nil {
		return fmt.Errorf("error during import: %w", err)
	}
	defer f.Close()

	switch t {
	case "node":
		nodes := struct {
			Nodes []*directory.TfgridNode2 `json:"nodes"`
		}{
			Nodes: []*directory.TfgridNode2{},
		}
		if err := json.NewDecoder(f).Decode(&nodes); err != nil {
			return err
		}

		err = db.Update(func(txn *badger.Txn) error {
			for _, node := range nodes.Nodes {
				fmt.Printf("import node %s\n", node.NodeID)
				b, err := json.Marshal(node)
				if err != nil {
					return err
				}
				if err := txn.Set(nodeKey(node.NodeID), b); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("error during import: %w", err)
		}
	case "farm":
		farms := struct {
			Farms []directory.TfgridFarm1 `json:"farms"`
		}{
			Farms: []directory.TfgridFarm1{},
		}
		if err := json.NewDecoder(f).Decode(&farms); err != nil {
			return err
		}

		err = db.Update(func(txn *badger.Txn) error {
			for _, farm := range farms.Farms {
				fmt.Printf("import farm %s\n", farm.Name)

				b, err := json.Marshal(farm)
				if err != nil {
					return err
				}

				if err := txn.Set(farmKey(farm.ID), b); err != nil {
					return err
				}
				if err := txn.Set([]byte(farm.Name), b); err != nil {
					return err
				}
			}
			return nil
		})
	case "reservation":
		reservations := struct {
			Reservations []*reservation `json:"reservations"`
		}{
			Reservations: []*reservation{},
		}
		if err := json.NewDecoder(f).Decode(&reservations); err != nil {
			return err
		}

		err = db.Update(func(txn *badger.Txn) error {
			for _, res := range reservations.Reservations {
				fmt.Printf("import reservation %s %s\n", res.Reservation.ID, res.NodeID)

				b, err := json.Marshal(res)
				if err != nil {
					return err
				}

				key := append(prefixReservation, []byte(res.Reservation.ID)...)
				if err := txn.Set(key, b); err != nil {
					return err
				}
			}
			return nil
		})
	default:
		return fmt.Errorf("type %s not supported", t)
	}

	return err
}
