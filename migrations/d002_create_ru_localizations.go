package migrations

import (
	"context"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/models"
)

func init() {
	Data.AddMigration("002_create_ru_localizations", d002{})
}

type d002 struct{}

func (m d002) Apply(ctx context.Context, db *gosql.DB) error {
	settingStore := models.NewSettingStore(db, "solve_setting", "solve_setting_event")
	localizations := [][2]string{
		{"account_missing_permissions", "Отсутсвуют необходимые права."},
		{"can_not_set_password", "Не удалось установить пароль."},
		{"code_is_empty", "Введен пустой код."},
		{"code_is_too_long", "Код слишком длинный."},
		{"compiler_not_found", "Компилятор не найден."},
		{"config_is_required", "Конфигурация является обязательной."},
		{"contest_not_found", "Соревнование не найдено."},
		{"duration_cannot_be_negative", "Продолжительность не может быть отрицательной."},
		{"email_has_invalid_format", "E-mail не соотвествует формату."},
		{"email_too_long", "E-mail слишком длинный."},
		{"email_too_short", "E-mail слишком короткий."},
		{"empty_problem_code", "Код задачи не указан."},
		{"file_is_required", "Файл является обязательным."},
		{"file_not_found", "Файл не найден."},
		{"first_name_too_long", "Имя слишком длинное."},
		{"first_name_too_short", "Имя слишком короткое."},
		{"form_has_invalid_field", "Форма содержит неправильные значения."},
		{"invalid_config", "Неправильная конфигурация."},
		{"invalid_contest_id", "Неправильный ID соревнования."},
		{"invalid_file_id", "Неправильный ID файла."},
		{"invalid_filter", "Неправильный фильтр."},
		{"invalid_form", "Неверная форма."},
		{"invalid_participant_id", "Неправильный ID участника."},
		{"invalid_password", "Неверный пароль."},
		{"invalid_problem_id", "Неправильный ID задачи."},
		{"invalid_scope_id", "Неправильный ID скоупа."},
		{"invalid_solution_id", "Неправильный ID решения."},
		{"invalid_user_id", "Неправильный ID пользователя."},
		{"last_name_too_long", "Фамилия слишком длинная."},
		{"last_name_too_short", "Фамилия слишком короткая."},
		{"login_has_invalid_format", "Логин не соотвествует формату."},
		{"login_too_long", "Логин слишком длинный"},
		{"login_too_short", "Логин слишком короткий"},
		{"mail_server_host_not_responding", "Почтовый сервер \"{host}\" не отвечает."},
		{"middle_name_too_long", "Отчество слишком длинное."},
		{"middle_name_too_short", "Отчество слишком короткое."},
		{"name_has_invalid_format", "Название не соотвествует формату."},
		{"name_is_required", "Имя является обязательным."},
		{"name_is_too_long", "Название слишком длинное."},
		{"name_is_too_short", "Название слишком короткое."},
		{"old_and_new_passwords_are_the_same", "Старый и новый пароли совпадают."},
		{"old_password_should_not_be_empty", "Старый пароль не может быть пустым."},
		{"participant_account_is_not_specified", "Аккаунт участника не указан."},
		{"participant_not_found", "Участник не найден."},
		{"participant_with_kind_kind_already_exists", "Участник с типом {kind} уже существует."},
		{"password_too_long", "Пароль слишком длинный."},
		{"password_too_short", "Пароль слишком короткий."},
		{"problem_code_does_not_exists", "Задача {code} не существует."},
		{"problem_id_already_exists", "Задача {id} уже добавлена."},
		{"problem_id_does_not_exists", "Задача {id} не существует."},
		{"problem_not_found", "Задача не найдена."},
		{"problem_with_code_code_already_exists", "Задача с кодом {code} уже добавлена."},
		{"role_role_already_exists", "Роль \"{role}\" уже существует."},
		{"role_role_already_has_child_child", "Роль \"{role}\" уже содержит роль \"{child}\"."},
		{"role_role_does_not_have_child_child", "Роль \"{role}\" не содержит роль \"{child}\"."},
		{"role_role_not_found", "Роль \"{role}\" не найдена."},
		{"scope_not_found", "Скоуп не найден."},
		{"session_not_found", "Сессия не найдена."},
		{"setting_key_cannot_be_empty", "Ключ настройки не может быть пустым."},
		{"setting_not_found", "Настройка не найдена."},
		{"solution_not_found", "Решение не найдено."},
		{"title_is_required", "Заголовок является обязательным."},
		{"title_is_too_long", "Заголовок слишком длинный."},
		{"title_is_too_short", "Заголовок слишком короткий."},
		{"unable_to_authorize", "Авторизация не пройдена."},
		{"unable_to_delete_builtin_role", "Не удалось удалить встроенную роль."},
		{"unknown_error", "Неизвестная ошибка."},
		{"user_id_does_not_exists", "Пользователь {id} не существует."},
		{"user_login_does_not_exists", "Пользователь \"{login}\" не существует."},
		{"user_not_found", "Пользователь не найден."},
		{"user_user_already_has_role_role", "Пользователь \"{user}\" уже имеет роль \"{role}\"."},
		{"user_user_does_not_have_role_role", "Пользователь \"{user}\" не имеет роли \"{role}\"."},
		{"user_with_login_login_already_exists", "Пользователь с логином \"{login}\" уже существует."},
	}
	for _, item := range localizations {
		setting := models.Setting{
			Key:   "localization.ru." + item[0],
			Value: item[1],
		}
		if err := settingStore.Create(ctx, &setting); err != nil {
			return err
		}
	}
	return nil
}

func (m d002) Unapply(ctx context.Context, db *gosql.DB) error {
	return nil
}
