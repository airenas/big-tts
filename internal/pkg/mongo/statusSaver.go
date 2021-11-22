package mongo

import (
	mng "github.com/airenas/async-api/pkg/mongo"
	"github.com/airenas/go-app/pkg/goapp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// StatusSaver saves process status to mongo db
type StatusSaver struct {
	SessionProvider *mng.SessionProvider
}

//NewStatusSaver creates StatusSaver instance
func NewStatusSaver(sessionProvider *mng.SessionProvider) (*StatusSaver, error) {
	f := StatusSaver{SessionProvider: sessionProvider}
	return &f, nil
}

// Save saves status to DB
func (ss *StatusSaver) Save(ID string, st, errStr string) error {
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
		bs = bson.M{"status": st, "error": errStr}
	}
	return mng.SkipNoDocErr(c.FindOneAndUpdate(ctx, bson.M{"ID": mng.Sanitize(ID)},
		bson.M{"$set": bs, "$unset": bu},
		options.FindOneAndUpdate().SetUpsert(true)).Err())
}
