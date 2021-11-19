package messages

//TagsType represents messages tags type
type TagsType int

const (
	Undefined TagsType = iota
	Created
	Filename
	Voice
	Speed
)

var (
	tagsName = map[TagsType]string{Undefined: "Undefined", Created: "Created",
		Filename: "Filename", Voice: "Voice",
		Speed: "Speed"}
	nameTags = map[string]TagsType{"Created": Created, "Filename": Filename,
		"Voice": Voice, "Speed": Speed}
)

func Name(st TagsType) string {
	return tagsName[st]
}

func From(st string) TagsType {
	return nameTags[st]
}
