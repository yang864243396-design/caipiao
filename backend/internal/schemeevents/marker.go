package schemeevents

type Marker interface {
	MarkScheme(memberID int64, instanceID string)
}
