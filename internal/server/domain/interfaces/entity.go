package interfaces

type Entity interface {
	GetID() string
	SetID(string)
	GetUserID() string
}
