package data

// User represents a sinle row from the accounts database
type User struct {
	Id       int64
	Username string
	RealName string
	GameYear int
}
