Где: Дашборд при выборе Customer и даты в календаре.
Проблема: Сервер режет фронтенд по заголовкам Content Security Policy(CSP). API отдает 200 ok, но браузер блокирует inline script и inline style(index.mjs).
Ошибки из консоли:
Executing inline script violates CSP directive 'script-src 'self''
Applying inline style violates CSP directive 'style-src 'self''

Где: Раздел Flows /flows через поиск по ctrl + k

1. Фатал фронтенда:
При переходе на /flows страница полностью падает в PAGE ERROR (app_error_boundary.tsx).
Ошибка: TypeError: crypto.randomUUID is not a function в flow_path_model.ts:17.
Причина: Код завязан на crypto.randomUUID(), который отключается браузером на http:// без HTTPS (Insecure Context). Нужен полифил/фоллбэк, либо принудительный редирект на HTTPS.
2. Ошибка 503 от бэка:
GET /api/v1/reports... отваливается со статусом 503 Service Unavailable.
3. CSP блокировки:
Сервер режет фронтенд по заголовкам Content Security Policy (CSP). API отдает 200 OK, но браузер блокирует inline script и inline style (index.mjs), из-за чего ломаются динамические элементы UI.
Ошибки:
Executing inline script violates CSP directive 'script-src 'self''
Applying inline style violates CSP directive 'style-src 'self''

Где: Campaigns -> Quick Create -> Выбор Customer Group

Фатал фронтенда при выборе группы клиентов:
При клике на селект выбора группы вылетает PAGE ERROR (app_error_boundary.tsx).
Ошибка: Minified React error #185 в select.tsx:172:7 и select.tsx:179:5.
Причина: Ошибка #185 означает Maximum update depth exceeded. Компонент Select входит в бесконечный цикл повторных рендеров (infinite re-render loop) при попытке обработать изменение значения или передаче функции в state. Сеть при этом чистая и отдает 200 OK.

Где: Campaigns -> Выбор кампании -> Кнопка Report

1. Нарушение клиентского роутинга SPA:
При клике на Report URL в адресной строке не меняется (/dashboards/campaign/id) и страница отчета не открывается. Переход происходит только после принудительного обновления страницы (Ctrl + R).
2. Ошибка клонирования (Network):
Запрос POST /api/v1/campaigns/clone отваливается со статусом 400 Bad Request.

Где: Campaigns -> Фильтр по статусу (табы All / Active / Paused / Archived)

Отсутствие реактивного обновления таблицы при фильтрации:
При клике на вкладку статуса (например, Paused 13) UI меняет активный таб, но список кампаний в таблице не фильтруется и продолжают отображаться все кампании (Active, Deleted, Paused). Вкладка Сеть при этом чистая и не отправляет повторный запрос за отфильтрованными данными. Отфильтрованный список появляется только после принудительной перезагрузки страницы (Ctrl + R).

Где: Campaigns -> Фильтр PACING

Фатал фронтенда при выборе фильтра Pacing:
При вызове выпадающего списка Pacing страница крашится в PAGE ERROR (app_error_boundary.tsx).
Ошибка: Minified React error #185 в select.tsx:172:7 и select.tsx:179:5.
Причина: Тот же зацикленный рендер компонентов (Maximum update depth exceeded) в компоненте Select при попытке раскрыть или выбрать вариант пейсинга. Сеть чистая.
Где: Campaigns -> Элементы управления фильтрацией (Owners, Country, Columns)

Фатал UI-компонентов Popover и DropdownMenu:
При открытии фильтров Owners, Country и селектора Columns интерфейс падает в PAGE ERROR (app_error_boundary.tsx).
Ошибки:
1. Фильтры Owners и Country: Minified React error #185 в popover.tsx:186:9 и popover.tsx:189:7.
2. Селектор Columns: Minified React error #185 в dropdown-menu.tsx:151:9 и dropdown-menu.tsx:154:7.
Сеть чистая

Где: Campaigns -> Кнопка контекстного меню [...] (три точки правее Archive)

Фатал DropdownMenu при вызове дополнительного меню действий:
При клике на три точки правее Archive страница падает в PAGE ERROR (app_error_boundary.tsx).
Ошибка: Minified React error #185 в dropdown-menu.tsx:151:9 и dropdown-menu.tsx:154:7.
Причина: Тот же зацикленный рендер (Maximum update depth exceeded) в компоненте DropdownMenu при открытии меню. Сеть чистая.

Где: Campaigns -> Фильтрация по статусам (All / Active / Paused / Archived / Warnings)

Системный баг реактивности таблицы при фильтрации:
При выборе любой вкладки статуса (Active, Paused, Archived, Warnings) кнопка статуса визуально переключается, но данные в таблице ниже не реагируют и продолжают отображать сплошной список абсолютно всех кампаний любого статуса. Запросы в сеть не отправляются. Полноценная фильтрация таблицы и сброс лишних строк происходят исключительно после ручного обновления страницы через Ctrl + R.

Где: Левое навигационное меню -> Навигация между Campaigns и Landing Pages (/landers)

Блокировка роутинга при переходе с раздела Campaigns:
При находящейся в фокусе странице Campaigns клик по любому другому пункту бокового меню (Landing Pages и т.д.) меняет URL в адресной строке браузера (например, на /landers), но сам интерфейс не перерисовывается и остается заблокированным на странице Campaigns. Консоль и Сеть молчат, новые запросы не отправляются. Если после клика сделать принудительный рестарт через Ctrl + R, целевая страница (/landers) успешно загружается. При этом с Landing Pages роутер работает корректно и позволяет переходить в другие разделы вплоть до повторного захода на Campaigns, после чего роутинг снова блокируется.