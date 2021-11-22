package status

//Status represents synthesize status
type Status int

const (
	//Uploaded value
	Uploaded Status = iota + 1
	Split
	Synthesize
	Join
	Completed
)

var (
	statusName = map[Status]string{Uploaded: "UPLOADED", Completed: "COMPLETED",
		Split: "Split", Synthesize: "Synthesize",
		Join: "Join"}
	nameStatus = map[string]Status{"UPLOADED": Uploaded, "COMPLETED": Completed,
		"Synthesize": Synthesize, "Join": Join,
		"Split": Split}
)

func (st Status) String() string {
	return statusName[st]
}

func From(st string) Status {
	return nameStatus[st]
}
