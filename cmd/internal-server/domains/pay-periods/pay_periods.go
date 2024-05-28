package payperiods

type Status int

const (
	Pending Status = iota
	Active
	Edit
)

func (s Status) String() string {
	switch s {
	case Pending:
		return "pending"

	case Active:
		return "active"

	case Edit:
		return "edit"
	}

	panic("invalid pay period status")
}
