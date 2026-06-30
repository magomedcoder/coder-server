package domain

import "time"

type MCPServer struct {
	ID             int64
	UserID         *int
	Name           string
	Enabled        bool
	Transport      string
	Command        string
	ArgsJSON       string
	EnvJSON        string
	URL            string
	HeadersJSON    string
	TimeoutSeconds int32
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
