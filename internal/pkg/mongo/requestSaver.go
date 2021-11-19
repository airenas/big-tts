package mongo

import (
	mng "github.com/airenas/async-api/pkg/mongo"
	"github.com/airenas/big-tts/internal/pkg/persistence"
	"github.com/airenas/go-app/pkg/goapp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// RequestSaver saves process request to mongo db
type RequestSaver struct {
	SessionProvider *mng.SessionProvider
}

//NewRequestSaver creates RequestSaver instance
func NewRequestSaver(sessionProvider *mng.SessionProvider) (*RequestSaver, error) {
	f := RequestSaver{SessionProvider: sessionProvider}
	return &f, nil
}

// Save saves resquest to DB
func (ss *RequestSaver) Save(data *persistence.ReqData) error {
	goapp.Log.Infof("Saving request %s: %s", data.ID, data.Email)

	c, ctx, cancel, err := mng.NewCollection(ss.SessionProvider, requestTable)
	if err != nil {
		return err
	}
	defer cancel()

	return mng.SkipNoDocErr(c.FindOneAndUpdate(ctx, bson.M{"ID": mng.Sanitize(data.ID)},
		bson.M{"$set": bson.M{"email": data.Email, "voice": data.Voice,
			"speed": data.Speed, "filename": data.Filename, "outputFormat": data.OutputFormat}},
		options.FindOneAndUpdate().SetUpsert(true)).Err())
}
