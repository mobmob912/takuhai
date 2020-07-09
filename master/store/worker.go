package store

import (
	"context"
	"log"

	"github.com/mobmob912/takuhai/domain"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mobmob912/takuhai/master/master/repository"
	"github.com/mobmob912/takuhai/master/worker"
	"go.mongodb.org/mongo-driver/mongo"
)

type workerStore struct {
	client *mongo.Client
}

func NewWorker(c *mongo.Client) repository.Worker {
	return &workerStore{
		client: c,
	}
}

const (
	workerCollection = "worker"
)

func (w *workerStore) Get(ctx context.Context, id string) (*worker.Worker, error) {
	wk := &worker.Worker{}
	collection := w.client.Database(databaseName).Collection(workerCollection)
	if err := collection.FindOne(ctx, bson.D{{"id", id}}).Decode(&wk); err != nil {
		return nil, err
	}
	return wk, nil
}

func (w *workerStore) ListAll(ctx context.Context) ([]*worker.Worker, error) {
	var ns []*worker.Worker
	collection := w.client.Database(databaseName).Collection(workerCollection)
	cur, err := collection.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		var n worker.Worker
		if err := cur.Decode(&n); err != nil {
			return nil, err
		}
		ns = append(ns, &n)
	}
	return ns, nil
}

func (w *workerStore) ListClouds(ctx context.Context) ([]*worker.Worker, error) {
	var ns []*worker.Worker
	collection := w.client.Database(databaseName).Collection(workerCollection)
	cur, err := collection.Find(ctx, bson.D{{"place", domain.PlaceCloud}})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		var n worker.Worker
		if err := cur.Decode(&n); err != nil {
			return nil, err
		}
		ns = append(ns, &n)
	}
	return ns, nil
}

func (w *workerStore) Set(ctx context.Context, id string, wf *worker.Worker) error {
	wf.ID = id
	log.Println(wf)
	collection := w.client.Database(databaseName).Collection(workerCollection)
	if _, err := collection.InsertOne(ctx, wf); err != nil {
		return err
	}
	return nil
}

func (w *workerStore) Update(ctx context.Context, id string, wk *worker.Worker) error {
	update := bson.D{{"$set", wk}}
	collection := w.client.Database(databaseName).Collection(workerCollection)
	if _, err := collection.UpdateOne(ctx, bson.D{{"id", id}}, update); err != nil {
		return err
	}
	return nil
}

func (w *workerStore) Delete(ctx context.Context, id string) error {
	collection := w.client.Database(databaseName).Collection(workerCollection)
	if _, err := collection.DeleteOne(ctx, bson.D{{"id", id}}); err != nil {
		return err
	}
	return nil
}
