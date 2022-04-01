package persistence

import "time"

type (

	//ReqData table
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
		RequestID    string `bson:"requestID,omitempty"`
	}

	//Status information table
	Status struct {
		ID     string `bson:"ID"`
		Status string `bson:"status,omitempty"`
		Error  string `bson:"error,omitempty"`
	}
)
