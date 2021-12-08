package mongo

import mng "github.com/airenas/async-api/pkg/mongo"

const (
	requestTable = "requests"
	statusTable  = "status"
	EmailTable   = "emailLock"
)

func GetIndexes() []mng.IndexData {
	return []mng.IndexData{
		mng.NewIndexData(requestTable, "ID", true),
		mng.NewIndexData(statusTable, "ID", true),
		mng.NewIndexData(EmailTable, "ID", false),
	}
}

func Tables() []string {
	return []string{requestTable, statusTable, EmailTable}
}
