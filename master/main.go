package main

import (
	"context"
	"log"

	"github.com/mobmob912/takuhai/master/uid"

	"github.com/mobmob912/takuhai/master/master"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/mobmob912/takuhai/master/store"

	"github.com/mobmob912/takuhai/master/api"
)

type Weight struct {
}
type Master interface {
}

func main() {
	log.SetFlags(log.Lshortfile)
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	mongoClient, err := mongo.NewClient(options.Client().ApplyURI("mongodb://root:example@localhost:27017"))
	if err != nil {
		return err
	}

	if err := mongoClient.Connect(context.Background()); err != nil {
		return err
	}

	nodeRepo := store.NewWorker(mongoClient)
	workflowRepo := store.NewWorkflow(mongoClient)
	uidGen := uid.NewUIDGenerator()
	sch := master.NewMaster(nodeRepo, workflowRepo, uidGen)

	if err := sch.Init(context.Background()); err != nil {
		return err
	}

	log.Println("serve")
	s := api.NewServer(sch)
	return s.Serve()
}
