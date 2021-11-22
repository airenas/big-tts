package mongo

import mng "github.com/airenas/async-api/pkg/mongo"

const (
	requestTable = "requests"
	statusTable  = "status"
)

func GetIndexes() []mng.IndexData {
	return []mng.IndexData{mng.NewIndexData(requestTable, "ID", true), mng.NewIndexData(statusTable, "ID", true)}
}
