package locker

var (
	locked = make(map[string]bool)
)

func Lock(key string) {
	locked[key] = true
}

func IsLocked(key string) bool {
	return locked[key]
}

func Unlock(key string) {
	delete(locked, key)
}
