package slice

const (
	defaultValue = -1
)

// MinSize возвращает минимальный размер слайса, учитывая limit и defaultLen
func MinSize(limit, defaultLen int) int {
	if limit != defaultValue && limit < defaultLen {
		return limit
	}

	return defaultLen
}
