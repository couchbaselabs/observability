package values

// User is a basic representation of a multi cluster user.
type User struct {
	User     string `json:"user"`
	Password []byte `json:"-"`
	Admin    bool   `json:"admin"`
}
