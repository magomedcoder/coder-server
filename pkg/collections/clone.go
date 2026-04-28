package collections

import "maps"

func CloneAnyMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}

	dst := make(map[string]any, len(src))
	maps.Copy(dst, src)

	return dst
}

func CloneStringSlice(src []string) []string {
	if len(src) == 0 {
		return nil
	}

	dst := make([]string, len(src))
	copy(dst, src)
	return dst
}
