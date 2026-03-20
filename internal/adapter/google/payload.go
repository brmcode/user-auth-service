package google

type Payload struct {
	Subject   string // Google's unique user ID (sub)
	Email     string
	FirstName string
	LastName  string
	Name      string
	AvatarURL string
}
