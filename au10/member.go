package au10

// Member describes an object that is a member of a group with access control.
type Member interface {
	// GetMembership returns user membership.
	GetMembership() Membership
}
