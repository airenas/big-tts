package persistence

import "time"

type (
	ReqData struct {
		ID           string
		Voice        string
		Filename     string
		SaveRequest  bool
		Speed        float64
		OutputFormat string
		Created      time.Time
		Email        string
		SaveTags     []string
	}

	Status struct {
		ID               string   `bson:"ID"`
		Status           string   `bson:"status,omitempty"`
		Error            string   `bson:"error,omitempty"`
	}
)
