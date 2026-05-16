package constants

type contextKey string

const (
	CodeNotFound      = "NOT_FOUND"
	MsgNotFoundDepart = "Подразделение не найдено"
	MsgNotFoundEmpl   = "Родительское подразделение не найдено"
	CodeInvalidInput  = "INVALID_INPUT"
	MsgInvalidInput   = "Неверные данные запроса"
	CodeConflict      = "CONFLICT"
	MsgConflict       = "Конфликт данных"
	CodeInternalError = "INTERNAL_ERROR"
	MsgInternalError  = "Внутренняя ошибка"
	CodeInvalidJson   = "INVALID_JSON"
	MsgInvalidJson    = "Ошибка парсинга JSON"
	CodeInvalidID     = "INVALID_ID"
	MsgInvalidID      = "ID не валидный"
	CodeInvalidDate   = "INVALID_DATE"
	MsgInvalidDate    = "Формат даты должен быть YYYY-MM-DD"

	LoggerKey = contextKey("logger")
	ID        = "id"
)
