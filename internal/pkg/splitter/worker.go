package splitter

import "github.com/airenas/go-app/pkg/goapp"

type Worker struct {
}

func NewWorker() (*Worker) {
	return &Worker{}
}

func (w *Worker) Do(ID string) error {
	goapp.Log.Infof("Doing job for %s", ID)
	return nil
}
