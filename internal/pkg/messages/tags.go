package messages

//TagsType represents messages tags type
type TagsType int

const (
	Undefined TagsType = iota
	Created
	Filename
	Voice
	Speed
	Format
	SaveRequest
	SaveTags
)

var (
	tagsName = map[TagsType]string{Undefined: "Undefined", Created: "Created",
		Filename: "Filename", Voice: "Voice",
		Speed: "Speed", Format: "Format", SaveRequest: "SaveRequest", SaveTags: "SaveTags"}
	nameTags = map[string]TagsType{"Created": Created, "Filename": Filename,
		"Voice": Voice, "Speed": Speed, "Format": Format, "SaveRequest": SaveRequest, "SaveTags": SaveTags}
)

func (st TagsType) Name() string {
	return tagsName[st]
}

func From(st string) TagsType {
	return nameTags[st]
}
