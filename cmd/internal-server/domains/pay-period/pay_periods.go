package payperiod

type Status int

const (
	Pending Status = iota
	Active
)

func (s Status) String() string {
	switch s {
	case Pending:
		return "pending"

	case Active:
		return "active"
	}

	panic("invalid worker status")
}
