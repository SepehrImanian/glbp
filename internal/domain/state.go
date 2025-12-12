package domain

type GatewayRole int

const (
	RoleListen GatewayRole = iota
	RoleStandby
	RoleAVG
)
