package mongo

import (
	mng "github.com/airenas/async-api/pkg/mongo"
	"github.com/airenas/big-tts/internal/pkg/persistence"
	"github.com/airenas/go-app/pkg/goapp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Status saves process status to mongo db
type Status struct {
	SessionProvider *mng.SessionProvider
}

//NewStatus creates Status instance
func NewStatus(sessionProvider *mng.SessionProvider) (*Status, error) {
	f := Status{SessionProvider: sessionProvider}
	return &f, nil
}

// Save saves status to DB
func (ss *Status) Save(ID string, st, errStr string) error {
	goapp.Log.Infof("Saving status %s: %s", ID, st)

	c, ctx, cancel, err := mng.NewCollection(ss.SessionProvider, statusTable)
	if err != nil {
		return err
	}
	defer cancel()
	bu := bson.M{}
	bs := bson.M{"status": st}
	if errStr == "" {
		bu = bson.M{"error": 1}
	} else {
		if st != "" {
			bs = bson.M{"status": st, "error": errStr}
		} else {
			bs = bson.M{"error": errStr}
		}
	}
	return mng.SkipNoDocErr(c.FindOneAndUpdate(ctx, bson.M{"ID": mng.Sanitize(ID)},
		bson.M{"$set": bs, "$unset": bu},
		options.FindOneAndUpdate().SetUpsert(true)).Err())
}

// Get retrieves status from DB
func (ss *Status) Get(id string) (*persistence.Status, error) {
	goapp.Log.Infof("Retrieving status %s", id)

	c, ctx, cancel, err := mng.NewCollection(ss.SessionProvider, statusTable)
	if err != nil {
		return nil, err
	}
	defer cancel()

	var m persistence.Status
	err = c.FindOne(ctx, bson.M{"ID": id}).Decode(&m)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &m, mng.SkipNoDocErr(err)
}
