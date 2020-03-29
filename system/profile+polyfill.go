// +build windows

package system

func StartProfile() Stopper {
	return noopStopper
}
