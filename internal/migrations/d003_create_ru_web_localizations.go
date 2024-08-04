package migrations

import (
	"context"
	"database/sql"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/internal/models"
)

func init() {
	Data.AddMigration("003_create_ru_web_localizations", d003{})
}

type d003 struct{}

func (m d003) Apply(ctx context.Context, db *gosql.DB) error {
	settingStore := models.NewSettingStore(db, "solve_setting", "solve_setting_event")
	localizations := [][2]string{
		{"actions", "Действия"},
		{"add", "Добавить"},
		{"admin", "Админ"},
		{"all", "Все"},
		{"author", "Автор"},
		{"change_language", "Сменить язык"},
		{"choose_file", "Выберите файл"},
		{"code", "Код"},
		{"compiler", "Компилятор"},
		{"contest_finished", "Завершено"},
		{"contest_not_started", "До начала"},
		{"contest_running", "Идёт"},
		{"contest", "Соревнование"},
		{"contests", "Соревнования"},
		{"create", "Создать"},
		{"description", "Описание"},
		{"duration", "Длительность"},
		{"edit_problem", "Редактировать задачу"},
		{"email_confirmation", "Вы получите письмо для подтверждения регистрации."},
		{"enter", "Войти"},
		{"first_name", "Имя"},
		{"forgot_password", "Забыли пароль?"},
		{"groups", "Группы"},
		{"hello", "Привет"},
		{"index", "Главная"},
		{"input_data", "Входные данные"},
		{"input", "Ввод"},
		{"interaction", "Протокол взаимодействия"},
		{"key", "Ключ"},
		{"language", "Язык"},
		{"last_name", "Фамилия"},
		{"login", "Войти"},
		{"logout", "Выйти"},
		{"manage", "Управление"},
		{"messages", "Сообщения"},
		{"middle_name", "Отчество"},
		{"name", "Имя"},
		{"new_message", "Новое сообщение"},
		{"new_question", "Новый вопрос"},
		{"notes", "Примечания"},
		{"official", "Официальное"},
		{"on_test", "на тесте"},
		{"or", "или"},
		{"output_data", "Выходные данные"},
		{"output", "Вывод"},
		{"page_not_found", "Страница не найдена"},
		{"participant_manager", "Менеджер"},
		{"participant_observer", "Наблюдатель"},
		{"participant_regular", "Участник"},
		{"participant_upsolving", "Дорешивание"},
		{"participant", "Участник"},
		{"participants", "Участники"},
		{"password", "Пароль"},
		{"passwords_do_not_match", "Пароли не совпадают"},
		{"paste_source_code", "вставить исходный код"},
		{"penalty", "Штраф"},
		{"points", "Баллы"},
		{"problem", "Задача"},
		{"problems", "Задачи"},
		{"profile", "Профиль"},
		{"question", "Вопрос"},
		{"regenerate_password", "Сгенерировать пароль"},
		{"register", "Регистрация"},
		{"repeat_password", "Повторите пароль"},
		{"repository", "Репозиторий"},
		{"roles", "Роли"},
		{"samples", "Примеры"},
		{"scope_users", "Пользователи скоупа"},
		{"scopes", "Скоупы"},
		{"score", "Счёт"},
		{"scoring", "Система оценки"},
		{"select_compiler", "Выберите компилятор"},
		{"select_problem", "Выберите задачу"},
		{"settings", "Настройки"},
		{"solution_file", "Файл решения"},
		{"solutions", "Решения"},
		{"source_code", "Исходный код"},
		{"standings", "Положение"},
		{"start", "Начало"},
		{"subject", "Тема"},
		{"submit_solution", "Отправить решение"},
		{"submit", "Отправить"},
		{"theme_dark", "Тёмная"},
		{"theme_light", "Светлая"},
		{"theme", "Тема"},
		{"this_page_does_not_exists", "Такая страница не существует."},
		{"time", "Время"},
		{"title", "Название"},
		{"unfrozen", "Размороженное"},
		{"update", "Обновить"},
		{"username_restrictions", "Вы можете использовать только латинские буквы, цифры, символы «_» и «-». Имя пользователя может начинаться только на латинскую букву и заканчиваться на латинскую букву или цифру."},
		{"username", "Имя пользователя"},
		{"users", "Пользователи"},
		{"value", "Значение"},
		{"verdict_accepted_description", "Полное решение"},
		{"verdict_ce_description", "Ошибка компиляции"},
		{"verdict_failed_description", "Не удалось протестировать"},
		{"verdict_mle_description", "Превышено ограничение памяти"},
		{"verdict_pa_description", "Частичное решение"},
		{"verdict_pe_description", "Неправильный формат вывода"},
		{"verdict_queued_description", "В очереди"},
		{"verdict_queued_title", "В очереди"},
		{"verdict_re_description", "Ошибка исполнения"},
		{"verdict_rejected_description", "Отклонено"},
		{"verdict_running_description", "Выполняется"},
		{"verdict_running_title_test", "Тест {test}"},
		{"verdict_running_title", "Выполняется"},
		{"verdict_tle_description", "Превышено ограничение времени"},
		{"verdict_wa_description", "Неправильный ответ"},
		{"verdict", "Вердикт"},
	}
	for _, item := range localizations {
		setting := models.Setting{
			Key:   "localization.ru.web." + item[0],
			Value: item[1],
		}
		if _, err := settingStore.FindOne(ctx, FindQuery{
			Where: gosql.Column("key").Equal(setting.Key),
		}); err != nil {
			if err != sql.ErrNoRows {
				return err
			}
		} else {
			continue
		}
		if err := settingStore.Create(ctx, &setting); err != nil {
			return err
		}
	}
	return nil
}

func (m d003) Unapply(ctx context.Context, db *gosql.DB) error {
	return nil
}
