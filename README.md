# Subscriptions Service

REST-сервис для агрегации данных об онлайн-подписках пользователей. Тестовое задание на позицию Junior Golang Developer (Effective Mobile).

## Стек

- **Go**: `net/http` + роутер [chi](https://github.com/go-chi/chi)
- **PostgreSQL**: драйвер [pgx](https://github.com/jackc/pgx)
- **Миграции**: [golang-migrate](https://github.com/golang-migrate/migrate), применяются автоматически при старте сервиса
- **Конфигурация**: `.env` через [cleanenv](https://github.com/ilyakaznacheev/cleanenv), в Docker переменные окружения
- **Логирование**: стандартный `log/slog`, формат и уровень зависят от окружения
- **Документация**: OpenAPI 3.0 + Swagger UI
- **Запуск**: Docker Compose

## API

| Метод | Путь | Описание |
|---|---|---|
| POST | `/subscriptions` | Создать запись о подписке |
| GET | `/subscriptions` | Список подписок с фильтрами |
| GET | `/subscriptions/{id}` | Получить подписку по ID |
| PUT | `/subscriptions/{id}` | Обновить подписку |
| DELETE | `/subscriptions/{id}` | Удалить подписку |
| GET | `/subscriptions/summary` | Суммарная стоимость подписок за период |
| GET | `/health` | Проверка работоспособности |

## Конфигурация

Все параметры в `.env.example`:

| Переменная | Описание | По умолчанию |
|---|---|---|
| `ENV` | Окружение: `local` (текстовые логи, Debug), `dev` (JSON, Debug), `prod` (JSON, Info) | `local` |
| `HTTP_PORT` | Порт HTTP-сервера | `8080` |
| `HTTP_READ_TIMEOUT` | Таймаут чтения запроса | `5s` |
| `HTTP_WRITE_TIMEOUT` | Таймаут записи ответа | `10s` |
| `HTTP_SHUTDOWN_TIMEOUT` | Таймаут graceful shutdown | `10s` |
| `DB_HOST` | Хост PostgreSQL | `localhost` |
| `DB_PORT` | Порт PostgreSQL | `5432` |
| `DB_USER` | Пользователь БД | `postgres` |
| `DB_PASSWORD` | Пароль БД | обязательно |
| `DB_NAME` | Имя базы данных | `subscriptions` |
| `DB_SSLMODE` | Режим SSL | `disable` |
| `MIGRATIONS_PATH` | Путь к каталогу миграций | `migrations` |
