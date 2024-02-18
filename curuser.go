package avfs

// CurUserMgr is the interface that wraps the current user related methods of a file system.
type CurUserMgr interface {
	// SetUser sets the current user.
	// If the user can't be changed an error is returned.
	SetUser(user UserReader) error

	// SetUserByName sets the current user by name.
	// If the user is not found, the returned error is of type UnknownUserError.
	SetUserByName(name string) error

	// User returns the current user.
	User() UserReader
}

// CurUserFn provides current user functions to a file system.
type CurUserFn struct {
	user UserReader
}

// SetUser sets the current user.
func (vst *CurUserFn) SetUser(user UserReader) error {
	vst.user = user

	return nil
}

// User returns the current user.
func (vst *CurUserFn) User() UserReader {
	return vst.user
}
