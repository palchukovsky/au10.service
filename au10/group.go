package au10

///////////////////////////////////////////////////////////////////////////////

// Group describes group entity.
type Group struct {
	Domain string
	Name   string
}

///////////////////////////////////////////////////////////////////////////////

// Rights describes an access rights group.
type Rights interface {
	// Get returns group info.
	Get() Group
}

// CreateRights creates a new instance of Rights.
func CreateRights(domain, name string) Rights {
	return &group{Group{Domain: domain, Name: name}}
}

///////////////////////////////////////////////////////////////////////////////

// Membership describes a membership object.
type Membership interface {
	// Get returns group info.
	Get() Group
	// IsAllowed returns true if one or more given groups are accessible for the membership.
	IsAllowed([]Rights) bool
}

// CreateMembership creates a new instance of Membership.
func CreateMembership(domain, name string) Membership {
	return &group{Group{Domain: domain, Name: name}}
}

///////////////////////////////////////////////////////////////////////////////

type group struct {
	group Group
}

func (group *group) Get() Group { return group.group }

func (group *group) IsAllowed(rights []Rights) bool {
	if group.group.Name == "*" && group.group.Domain == "*" {
		return true
	}
	for _, right := range rights {
		rightsGroup := right.Get()
		if !group.isItemAvailable(group.group.Name, rightsGroup.Name) {
			continue
		}
		if !group.isItemAvailable(group.group.Domain, rightsGroup.Domain) {
			continue
		}
		return true
	}
	return false
}

func (*group) isItemAvailable(membership, rights string) bool {
	return membership == "*" || rights == "*" || membership == rights
}

///////////////////////////////////////////////////////////////////////////////
