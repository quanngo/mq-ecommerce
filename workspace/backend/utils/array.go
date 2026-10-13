package utils

func CleanDuplicated[T comparable](slices []T) []T {
	allKeys := make(map[T]bool)
	list := []T{}
	for _, item := range slices {
		// Check if the item has not been encountered before.
		if _, value := allKeys[item]; !value {
			// If the item is unique, add it to the map and the list.
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}
