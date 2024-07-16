package earnings

type Event int

const (
	Created Event = iota
	Updated
	Deleted
)

func (s Event) String() string {
	switch s {
	case Created:
		return "Created"

	case Updated:
		return "Updated"

	case Deleted:
		return "Deleted"
	}

	panic("invalid earning event")
}
