package entities

type Request interface {
	Marshal() ([]byte, error)
}
