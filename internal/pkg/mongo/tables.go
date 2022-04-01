package mongo

import mng "github.com/airenas/async-api/pkg/mongo"

const (
	// RequestTable is a name for requests
	RequestTable = "requests"
	statusTable  = "status"
	// EmailTable is name for email lock table
	EmailTable = "emailLock"
)

// GetIndexes returns indexes for mongo tables
func GetIndexes() []mng.IndexData {
	return []mng.IndexData{
		mng.NewIndexData(RequestTable, "ID", true),
		mng.NewIndexData(statusTable, "ID", true),
		mng.NewIndexData(EmailTable, "ID", false),
	}
}

// Tables returns tables for system
func Tables() []string {
	return []string{RequestTable, statusTable, EmailTable}
}
