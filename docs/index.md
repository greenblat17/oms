# Домашнее задание №3

- Проанализируйте запросы в БД. Приложите результаты анализа в README.md. Добавьте индексы, где это необходимо.

## Анализ запросов

### База данных

Для проведения анализа было добавлено несколько миллионов строк:

```sql
select count(*) from orders
```

<img src="assets/count.png">

### Запрос FindOrdersByRecipientID

```sql
SELECT id, recipient_id, storage_until, issued_at, returned_at, hash
FROM orders
WHERE recipient_id = 1
ORDER BY storage_until DESC;
```

#### Изначально

<img src="assets/query1_explain_before1.png">
<img src="assets/query1_explain_before5.png">
<img src="assets/query1_explain_before9.png">

Параллельное последовательное сканирование (Parallel Seq Scan)
отбрасывает значительное количество строк, большинство строк в таблице
не соответствуют критерию `recipient_id = 1`.

Однако запрос имеет высокую стоимость и время выполнения. Внешняя сортировка (external merge)
указывает на недостаток памяти, что приводит к использованию диска и замедляет выполнение запроса.
начительное количество строк удаляется в процессе фильтрации (Rows Removed by Filter: 595941) - необходимо
улучшить индексирование запроса

#### Рекомендации

<img src="assets/query1_recommendations_before5.png">
(Для остальных запросов рекомендации были аналогичные)

- Так как фильтрация происходит по полю `recipient_id` имеет смысл создать
  BTREE индекс для этого поля

```sql
CREATE INDEX idx_recipient_id ON public.orders 
    USING BTREE (recipient_id);
```

- Увеличить значение work_mem

```sql
SET work_mem = '64MB';
```

#### Результат

<img src="assets/query1_result1.png">
<img src="assets/query1_result5.png">
<img src="assets/query1_result9.png">

Использование индекса `idx_recipient_id` ускоряет поиск по сравнению
с полным сканированием таблицы

#### Сравнение с составным индексом

Запрос выполняется достаточно быстро, однако затраты все еще высоки.
Составной индекс может ускорить выполнение запроса, исключив необходимость сортировки на этапе
выполнения запроса.

```sql
CREATE INDEX idx_recipient_id_storage_until ON orders (recipient_id, storage_until DESC);
```

<img src="assets/query1_comparison.png">

В целом оба индекса дают примерно один и тот же результат

### Запрос FindReturnedOrdersWithPagination

```sql
SELECT id, recipient_id, storage_until, issued_at, returned_at, hash
FROM orders
WHERE returned_at IS NOT NULL
ORDER BY storage_until DESC
LIMIT 10
OFFSET 0;
```

#### Изначально

<img src="assets/query2_explain_before.png">

Несмотря на использование параллельного сканирования, полное последовательное сканирование
таблицы остается дорогим по времени
Использование top-N heapsort при сортировке по полю storage_until DESC помогает
в оптимизации памяти, но объем данных всё равно остается большим.

#### Рекомендации

<img src="assets/query2_recommendations.png">

Создание составного или частичного индекса

##### Составной индекс

Составной индекс будет включать оба столбца returned_at и storage_until.
Это ускорит фильтрацию по returned_at и сортировку по storage_until.

Индекс используется для фильтрации и сортировки одновременно, что может значительно ускорить
выполнение запроса. Однако Индекс будет занимать больше места, так как он будет включать все строки,
а не только те, где returned_at IS NOT NULL.

```sql
CREATE INDEX idx_returned_storage ON orders (returned_at, storage_until DESC);
```

##### Частичный индекс

Частичный индекс будет включать только строки, где returned_at IS NOT NULL, и индексировать только
столбец storage_until. Это ускорит запросы, которые фильтруют по returned_at IS NOT NULL и сортируют
по storage_until.

Индекс будет занимать меньше места, так как он будет включать только строки, удовлетворяющие условию
returned_at IS NOT NULL.

```sql
CREATE INDEX idx_partial_storage_until ON orders (storage_until DESC) WHERE returned_at IS NOT NULL;
```

##### Вывод

Так как в данном sql-запросе идет фильтрация именно по условию returned_at IS NOT NULL и сортировка по
storage_until частичный индекс больше подходит. Индекс будет сразу отсортирован по storage_until DESC

#### Результат

<img src="assets/query2_result.png">

Индекс idx_partial_storage_until эффективно используется для сортировки и ограничения результата
запроса. Это видно из низкой стоимости и быстрого времени выполнения.

#### Сравнение с BTREE

<img src="assets/query2_comparison.png">

Несмотря на параллельное выполнение, общий план имеет высокую стоимость и время выполнения.
Значительное количество строк удаляется в процессе фильтрации (Rows Removed by Filter: 328483) -
это указывает на то, что необходимо улучшить индексирование запроса

### Остальные запросы

В остальных запросах фильтрация используется по полю `id`, для которого и так создан
индекс, так как он является первичным ключом

<img src="assets/default_idx.png">

### Итоговый результат

```sql
SELECT indexname, indexdef
FROM pg_indexes
WHERE tablename = 'orders';
```

<img src="assets/result_idx.png">