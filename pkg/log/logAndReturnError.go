package log

func LogAndReturnError(err error) error {
	return Get().LogAndReturnError(err)
}
