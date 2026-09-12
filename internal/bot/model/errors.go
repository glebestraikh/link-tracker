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
	ErrLinkAlreadyTracked = newAppError("ссылка уже отслеживается")

	ErrChatNotFound = newAppError("чат не существует")

	ErrLinkOrChatNotFound = newAppError("чат не существует или ссылка не найдена")

	ErrChatAlreadyRegistered = newAppError("чат уже зарегистрирован")

	ErrInvalidUpdate = newAppError("некорректное обновление")

	ErrDeliveryFailed = newAppError("не удалось доставить обновление")

	ErrScrapperUnavailable = newAppError("сервис временно недоступен, попробуйте позже")

	ErrInternal = newAppError("внутренняя ошибка, попробуйте позже")
)
