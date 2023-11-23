package perms

type Permissions interface {
	HasPermission(name string) bool
}

type PermissionSet map[string]struct{}

func (p PermissionSet) AddPermission(names ...string) {
	for _, name := range names {
		p[name] = struct{}{}
	}
}

func (p PermissionSet) HasPermission(name string) bool {
	_, ok := p[name]
	return ok
}

func (p PermissionSet) Clone() PermissionSet {
	clone := PermissionSet{}
	for key := range p {
		clone[key] = struct{}{}
	}
	return clone
}
