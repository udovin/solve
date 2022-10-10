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
	localizations := map[string]string{
		"account_missing_permissions":               "Отсутсвуют необходимые права.",
		"code_is_empty":                             "Введен пустой код.",
		"code_is_too_long":                          "Код слишком длинный.",
		"compiler_not_found":                        "Компилятор не найден.",
		"contest_not_found":                         "Соревнование не найдено.",
		"duration_cannot_be_negative":               "Продолжительность не может быть отрицательной.",
		"empty_problem_code":                        "Код задачи не указан.",
		"form_has_invalid_field":                    "Форма содержит неправильные значения.",
		"invalid_config":                            "Неправильная конфигурация.",
		"invalid_contest_id":                        "Неправильный ID соревнования.",
		"invalid_filter":                            "Неправильный фильтр.",
		"invalid_participant_id":                    "Неправильный ID участника.",
		"invalid_solution_id":                       "Неправильный ID решения.",
		"name_is_too_long":                          "Название слишком длинное.",
		"name_is_too_short":                         "Название слишком короткое.",
		"participant_account_is_not_specified":      "Аккаунт участника не указан.",
		"participant_not_found":                     "Участник не найден.",
		"participant_with_kind_kind_already_exists": "Участник с типом {kind} уже существует.",
		"problem_code_does_not_exists":              "Задача {code} не существует.",
		"problem_id_already_exists":                 "Задача {id} уже добавлена.",
		"problem_id_does_not_exists":                "Задача {id} не существует.",
		"problem_with_code_code_already_exists":     "Задача с кодом {code} уже добавлена.",
		"title_is_required":                         "Заголовок является обязательным.",
		"title_is_too_long":                         "Заголовок слишком длинный.",
		"title_is_too_short":                        "Заголовок слишком короткий.",
		"user_id_does_not_exists":                   "Пользователь {id} не существует.",
		"user_login_does_not_exists":                "Пользователь \"{login}\" не существует.",
	}
	for key, localization := range localizations {
		setting := models.Setting{
			Key:   "localization.ru." + key,
			Value: localization,
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
