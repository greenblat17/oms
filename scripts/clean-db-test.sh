#!/bin/bash

# Переход в папку с текущим скриптом (если необходимо)
cd "$(dirname "$0")"

# Загружаем переменные из .env файла
if [[ -f ../.env ]]; then
    set -o allexport
    source ../.env
    set +o allexport
else
    echo "Ошибка: Файл .env не найден."
    exit 1
fi

# Проверка наличия файла конфигурации test.yml
CONFIG_FILE=../configs/test.yml
if [[ ! -f $CONFIG_FILE ]]; then
    echo "Ошибка: Файл конфигурации $CONFIG_FILE не найден."
    exit 1
fi

# Получаем значения из config.yml с помощью yq
DB_USERNAME=$(yq e '.db.username' $CONFIG_FILE)
DB_HOST=$(yq e '.db.host' $CONFIG_FILE)
DB_PORT=$(yq e '.db.port' $CONFIG_FILE)
DB_NAME=$(yq e '.db.db_name' $CONFIG_FILE)
SSL_MODE=$(yq e '.db.ssl_mode' $CONFIG_FILE)

# Заменяем переменные окружения в строке подключения
DB_CONNECTION_STRING="postgres://${DB_USERNAME}:${TEST_DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${SSL_MODE}"

SQL_QUERY="TRUNCATE TABLE orders;"

# Подключаемся к базе данных и выполняем SQL-запрос
psql "${DB_CONNECTION_STRING}" -c "${SQL_QUERY}"

# Проверка статуса выполнения команды
if [ $? -eq 0 ]; then
  echo "Database cleaned successfully."
else
  echo "Failed to clean the database."
  exit 1
fi