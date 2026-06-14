// Русские подписи для закрытого списка проблем (enum инструмента
// recommend_fertilizer на бэкенде). Ключи должны совпадать с
// llm.FertilizerProblemKeys — список приходит с сервера, подписи локальные.
export const PROBLEM_LABELS: Record<string, string> = {
  leaf_yellowing: 'Пожелтение листьев',
  leaf_spots: 'Пятна на листьях',
  wilting: 'Увядание',
  stunted_growth: 'Замедленный рост',
  poor_fruiting: 'Плохое плодоношение',
  root_rot: 'Корневая гниль',
  pest_aphid: 'Тля',
  pest_mite: 'Клещ',
  nitrogen_deficiency: 'Дефицит азота',
  phosphorus_deficiency: 'Дефицит фосфора',
  potassium_deficiency: 'Дефицит калия',
  calcium_deficiency: 'Дефицит кальция',
  magnesium_deficiency: 'Дефицит магния',
  iron_deficiency: 'Дефицит железа',
  general_stress: 'Общий стресс растения',
};

export function problemLabel(key: string): string {
  return PROBLEM_LABELS[key] ?? key;
}

// Подписи действий для журнала аудита.
export const ACTION_LABELS: Record<string, string> = {
  admin_setup: 'Создан аккаунт администратора',
  admin_login: 'Вход в панель',
  admin_login_failed: 'Неудачная попытка входа',
  admin_reset_requested: 'Запрошено восстановление пароля',
  admin_password_reset: 'Пароль восстановлен',
  admin_password_changed: 'Пароль изменён',
  admin_session_revoked: 'Сессия завершена',
  product_created: 'Товар добавлен',
  product_updated: 'Товар изменён',
  product_deleted: 'Товар удалён',
  product_image_uploaded: 'Загружено фото товара',
};

export function actionLabel(key: string): string {
  return ACTION_LABELS[key] ?? key;
}
