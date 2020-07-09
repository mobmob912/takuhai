package store

import (
	"context"

	"github.com/mobmob912/takuhai/domain"
	"github.com/mobmob912/takuhai/master/master/repository"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type workflow struct {
	client *mongo.Client
}

const (
	workflowCollection = "workflow"
)

func NewWorkflow(c *mongo.Client) repository.Workflow {
	return &workflow{
		client: c,
	}
}

func (w *workflow) Get(ctx context.Context, id string) (*domain.Workflow, error) {
	var wf domain.Workflow
	collection := w.client.Database(databaseName).Collection(workflowCollection)
	if err := collection.FindOne(ctx, bson.D{{"id", id}}).Decode(&wf); err != nil {
		return nil, err
	}
	return &wf, nil
}

func (w *workflow) GetByName(ctx context.Context, name string) (*domain.Workflow, error) {
	var wf domain.Workflow
	collection := w.client.Database(databaseName).Collection(workflowCollection)
	if err := collection.FindOne(ctx, bson.D{{"name", name}}).Decode(&wf); err != nil {
		return nil, err
	}
	return &wf, nil
}

func (w *workflow) GetStep(ctx context.Context, workflowID, stepID string) (*domain.Step, error) {
	var wf domain.Workflow
	collection := w.client.Database(databaseName).Collection(workflowCollection)
	if err := collection.FindOne(ctx, bson.D{{"id", workflowID}}).Decode(&wf); err != nil {
		return nil, err
	}
	for _, s := range wf.Steps {
		if s.ID == stepID {
			return s, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (w *workflow) ListAll(ctx context.Context) ([]*domain.Workflow, error) {
	var wfs []*domain.Workflow
	collection := w.client.Database(databaseName).Collection(workflowCollection)
	cur, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	for cur.Next(ctx) {
		var wf domain.Workflow
		if err := cur.Decode(&wf); err != nil {
			return nil, err
		}
		wfs = append(wfs, &wf)
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}
	return wfs, nil
}

func (w *workflow) CheckExistByByName(ctx context.Context, name string) (bool, error) {
	collection := w.client.Database(databaseName).Collection(workflowCollection)
	err := collection.FindOne(ctx, bson.M{"name": name}).Err()
	if err == mongo.ErrNoDocuments {
		return false, nil
	}
	return true, err
}

func (w *workflow) Set(ctx context.Context, id string, wf *domain.Workflow) error {
	wf.ID = id
	collection := w.client.Database(databaseName).Collection(workflowCollection)
	if collection.FindOne(ctx, bson.D{{"id", id}}).Err() == mongo.ErrNoDocuments {
		if _, err := collection.InsertOne(ctx, wf); err != nil {
			return err
		}
	} else {
		if _, err := collection.UpdateOne(ctx, bson.D{{"id", id}}, bson.D{{"$set", wf}}); err != nil {
			return err
		}
	}
	return nil
}
