package sythesizer

import (
	"github.com/pkg/errors"
)

type Worker struct {
}

func NewWorker() (*Worker, error) {
	res := &Worker{}
	return res, nil
}

func (w *Worker) Do(ID string) error {
	return errors.New("not implemented")
}
