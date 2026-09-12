package model

type AppError struct {
	name string
}

func (e AppError) Error() string {
	return e.name
}

func newAppError(name string) error {
	return AppError{
		name: name,
	}
}

var (
	ErrChatAlreadyExists = newAppError("чат уже существует")

	ErrChatNotFound = newAppError("чат не существует")

	ErrLinkAlreadyTracked = newAppError("ссылка уже отслеживается")

	ErrLinkNotFound = newAppError("ссылка не найдена")

	ErrTagNotFound = newAppError("тег не найден")

	ErrTagAlreadyExists = newAppError("тег уже существует")
)
