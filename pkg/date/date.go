package date

import "time"

const (
	dateLayout = "02.01.2006"
)

// ParseDateToUTC преобразует строку даты в UTC time.Time
func ParseDateToUTC(dateStr string) (time.Time, error) {
	t, err := time.Parse(dateLayout, dateStr)
	if err != nil {
		return time.Time{}, err
	}

	t = t.UTC()

	return t, nil
}

// ConvertUTCToStr преобразует time.Time в строку даты в формате "02.01.2006"
func ConvertUTCToStr(time time.Time) string {
	return time.Format(dateLayout)
}
