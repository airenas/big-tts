package mongo

import (
	"fmt"

	mng "github.com/airenas/async-api/pkg/mongo"
	"github.com/airenas/big-tts/internal/pkg/persistence"
	"github.com/airenas/big-tts/internal/pkg/status"
	"github.com/airenas/go-app/pkg/goapp"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	mgodr "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Request provides request persistence
type Request struct {
	SessionProvider *mng.SessionProvider
	statusSaver     *Status
}

//NewRequest creates Request instance
func NewRequest(sessionProvider *mng.SessionProvider) (*Request, error) {
	f := Request{SessionProvider: sessionProvider, statusSaver: &Status{SessionProvider: sessionProvider}}
	return &f, nil
}

// Save saves resquest to DB
func (rm *Request) Save(data *persistence.ReqData) error {
	goapp.Log.Infof("Saving request %s: %s", data.ID, goapp.Sanitize(data.Email))

	c, ctx, cancel, err := mng.NewCollection(rm.SessionProvider, RequestTable)
	if err != nil {
		return err
	}
	defer cancel()

	err = mng.SkipNoDocErr(c.FindOneAndUpdate(ctx, bson.M{"ID": mng.Sanitize(data.ID)},
		bson.M{"$set": bson.M{"email": data.Email, "voice": data.Voice,
			"speed": data.Speed, "filename": data.Filename, "outputFormat": data.OutputFormat,
			"saveRequest": data.SaveRequest}},
		options.FindOneAndUpdate().SetUpsert(true)).Err())
	if err != nil {
		return err
	}
	return rm.statusSaver.Save(data.ID, status.Uploaded.String(), "")
}

//GetResultFile returns file name by ID
func (rm *Request) GetResultFile(id string) (string, error) {
	goapp.Log.Infof("Getting file name by ID %s", goapp.Sanitize(id))
	m, err := rm.loadData(id)
	if err != nil {
		return "", err
	}
	if m.OutputFormat == "" {
		return "", errors.New("no output format")
	}
	return fmt.Sprintf("%s/result/result.%s", id, m.OutputFormat), nil
}

func (rm *Request) loadData(id string) (*persistence.ReqData, error) {
	c, ctx, cancel, err := mng.NewCollection(rm.SessionProvider, RequestTable)
	if err != nil {
		return nil, err
	}
	defer cancel()

	var res persistence.ReqData
	err = c.FindOne(ctx, bson.M{"ID": mng.Sanitize(id)}).Decode(&res)
	if err == mgodr.ErrNoDocuments {
		return nil, errors.Wrap(err, "no request by ID")
	}
	if err != nil {
		return nil, errors.Wrap(err, "can't get request record")
	}
	return &res, nil
}

//GetEmail returns email by ID
func (rm *Request) GetEmail(id string) (string, error) {
	goapp.Log.Infof("Getting email by ID %s", id)
	m, err := rm.loadData(id)
	if err != nil {
		return "", err
	}
	if m.Email == "" {
		return "", errors.New("no email")
	}
	return m.Email, nil
}
