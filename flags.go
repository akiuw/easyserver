package easyserver

type IServerFlags interface {
	GetPath() string
	GetConfigType() string
	GetRoles() []string
}
