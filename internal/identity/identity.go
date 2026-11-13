package identity

import gonanoid "github.com/matoous/go-nanoid/v2"

// GenerateRandomDeviceId returns a random string that could be used as device ID.
// For now it's a wrapper around gonanoid.Must(), but might change later
func GenerateRandomDeviceId() string {
	return gonanoid.Must()
}
