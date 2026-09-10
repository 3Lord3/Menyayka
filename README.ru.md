# Меняйка (Menyayka)

[English](README.md) | [Русский](README.ru.md)

Веб-сервис многостороннего бартера: пользователь выставляет товар и описывает, что хочет получить взамен; система ищет циклические цепочки обмена между несколькими участниками (2–5 человек), предлагает варианты, собирает отклики и проводит сделку через мягкую и жёсткую блокировку заявок.

---

## Содержание

1. [Описание проекта](#описание-проекта)
   - [Безопасность сделки](#безопасность-сделки)
   - [Ранжирование (LightGBM)](#ранжирование-lightgbm)
2. [Инструкция по запуску](#инструкция-по-запуску)
   - [Весь проект одной командой](#весь-проект-одной-командой)
   - [Разработка фронта без Docker](#разработка-фронта-без-docker-опционально)
   - [Полезные команды](#полезные-команды)
   - [CI/CD и деплой](#cicd-и-деплой)
3. [Технологии](#технологии)
4. [Структура проекта](#структура-проекта)
5. [Backend](#backend)
   - [Embeddings (TEI)](#embeddings-tei)
   - [Справочник API](#справочник-api)
6. [Frontend](#frontend)
7. [Команда](#команда)

---

## Описание проекта

Ключевые пользовательские сценарии:

- регистрация / вход / восстановление пароля;
- каталог своих товаров;
- заявки на обмен (что отдаю + что хочу);
- просмотр найденных цепочек-кандидатов;
- быстрая замена участника при отказе;
- отслеживание отправки и получения товара.

Как работает matching и построение цепочек:

1. Заявка получает embeddings (offer + want) через TEI (или stub в тестах).
2. Похожие заявки группируются в кластеры (cosine + категорийный отсев + direction margin).
3. Между кластерами строится граф; DFS ищет простые циклы длины 2–5.
4. Найденные циклы сохраняются как `CANDIDATE`, дедуплицируются и скорятся.
5. Участники ставят направленные votes; при замыкании кольца цепочка → `PROPOSED` (мягкая блокировка, дедлайн ~24ч).
6. Второй раунд подтверждений → `FROZEN` (жёсткая блокировка товаров/заявок).
7. При отказе — откат или быстрая замена участника из того же кластера.
8. Дальше — отслеживание отправки/получения до завершения обмена.

### Безопасность сделки

Чтобы участник цепочки не отдал свою вещь и не остался без встречного товара, обмен проходит через ПВЗ по модели группового хранения.

После подтверждения цепочки каждый участник обязан передать свой товар в ПВЗ до установленного срока. При приёме товар проверяется на соответствие характеристикам заявки. До завершения проверки всех товаров вещи хранятся в ПВЗ и не выдаются новым владельцам.

Цепочка готова к выдаче только когда все участники сдали товары и каждый товар прошёл проверку. После этого все вещи одновременно становятся доступны получателям.

Если кто-то не передал товар вовремя или товар не соответствует заявке, цепочка отменяется. Уже принятые в ПВЗ товары возвращаются первоначальным владельцам. Так никто не теряет свою вещь, пока не подтверждено, что вся цепочка может быть завершена.

### Ранжирование (LightGBM)

Кандидаты упорядочиваются по вероятности успешного завершения. Раньше порядок задавала линейная формула; её заменила модель LightGBM. Формула не учитывала порог слабого ребра, насыщение ликвидности и рост штрафа за длину цепочки быстрее линейного.

Модель обучена на синтетическом симуляторе (`ml/simulator`), который воспроизводит полную воронку сделки. На выходе получается *калиброванная* вероятность успеха.

Синтетика нужна из-за холодного старта: истории реальных обменов нет. Симулятор зашивает предметные законы в латентные переменные, модель видит только шумные наблюдаемые фичи — те же, что в продакшене. Утечек целевой переменной нет, `ml/features.json` зеркалит Go-экстрактор фич.

---

## Инструкция по запуску

### Весь проект одной командой

Нужны Docker + Docker Compose (Make опционален; можно вызывать `docker compose` напрямую).

```bash
cp .env.example .env   # на Apple Silicon добавь TEI_IMAGE=...cpu-arm64-latest (см. .env.example)
docker compose up -d --build
# или: make up
```

Поднимается всё: PostgreSQL + pgvector, миграции, MinIO, TEI, backend и frontend.

| Что | URL |
|-----|-----|
| UI (frontend) | http://localhost:3000 |
| API (backend) | http://localhost:8080 |
| MinIO | http://localhost:9000 |

Фронт отдаёт статику и проксирует `/api` → backend, `/s3` → MinIO (same-origin, пустой `VITE_API_BASE_URL`).

Остановка:

```bash
make down        # контейнеры + данные БД/MinIO; кэш TEI сохраняется
make down-all    # полный сброс, включая кэш модели
```

### Разработка фронта без Docker (опционально)

```bash
docker compose up -d db migrate minio minio-init tei app
cd frontend && npm ci && npm run dev   # http://localhost:5173 → proxy на :8080
```

### Полезные команды

```bash
make logs            # логи backend + frontend
make db-logs         # логи БД
make test            # unit-тесты backend
make run             # запуск backend на хосте (нужна поднятая БД)
make test-e2e        # E2E API (лёгкий compose без TEI)
make loadtest        # k6: register → item → list → login
```

### CI/CD и деплой

На pull request GitHub Actions запускает unit-тесты backend, проверки frontend и E2E. При пуше в `main` те же проверки плюс обязательный E2E проходят до сборки образов в GHCR и деплоя.

Продакшен-стек: `docker-compose.prod.yml` + Caddy + образы из GHCR. Автодеплой на VPS при пуше в `main` (workflow `.github/workflows/deploy.yml`): сборка и публикация образов в GHCR → SSH-деплой на VPS (migrate + `docker compose up`) → Caddy отдаёт `menyayka.tech`.

---

## Технологии

### Backend

| Компонент | Технология |
|-----------|------------|
| Язык | Go 1.26 |
| HTTP | Gin |
| БД | PostgreSQL 18 + pgvector 0.8.6 |
| Миграции | golang-migrate |
| Auth | JWT (Bearer) |
| Embeddings | Hugging Face TEI (`multilingual-e5-small`) / stub |
| Объектное хранилище | MinIO (S3-совместимое) |
| Почта | SMTP (восстановление пароля) |
| Тесты | `testing`, httptest; E2E с build-tag `e2e` |
| Нагрузка | k6 |

### Frontend

| Компонент | Технология |
|-----------|------------|
| UI | React 19 + TypeScript |
| Сборка | Vite |
| Архитектура | Feature-Sliced Design (FSD) |
| Состояние | Redux Toolkit + TanStack Query |
| UI-kit | Ant Design |
| Роутинг | React Router |
| HTTP | Axios |
| Качество | ESLint, Prettier, Vitest, Testing Library |

### Инфраструктура

| Компонент | Технология |
|-----------|------------|
| Локальный запуск | Docker Compose, Makefile |
| Reverse proxy / TLS | Caddy |

---

## Структура проекта

```
menyayka/
├── backend/
│   ├── cmd/api/                 # composition root, HTTP-сервер
│   ├── internal/
│   │   ├── api/                 # единый формат ответов/ошибок
│   │   ├── core/                # config, router, database, logger
│   │   ├── entity/              # доменные сущности и sentinel-ошибки
│   │   ├── handler/             # HTTP-слой (user, item, exchange_offer, chain)
│   │   ├── middleware/          # JWT auth
│   │   ├── repository/          # доступ к БД по фичам (chain, cluster, item,
│   │   │                        #   search, exchange_offer, user, …)
│   │   ├── service/             # бизнес-логика (matching, clustering, chain, …)
│   │   └── infrastructure/      # TEI, mailer, reservation, chain notification
│   ├── migrations/              # SQL up/down (golang-migrate)
│   ├── docs/                    # сгенерированный Swagger-спек + docs.go (swag init)
│   ├── pkg/                     # token, storage, validator, codestore, ranker
│   │   └── utils/ranker/models/ # экспорт модели LightGBM (text v3) + Go-инференс
│   └── tests/                   # unit / handler / service / repository / e2e
├── frontend/
│   ├── src/
│   │   ├── app/                 # провайдеры, router, store, темы
│   │   ├── pages/               # экраны
│   │   ├── features/            # auth, items, exchange-requests, chains, deal
│   │   ├── entities/            # модели и API сущностей
│   │   └── shared/              # UI, api-клиент, утилиты
│   ├── Dockerfile
│   ├── nginx.conf               # prod: только статика (Caddy спереди)
│   └── nginx.local.conf         # local compose: статика + /api + /s3
├── ml/
│   ├── simulator/               # генератор синтетической воронки (Go)
│   ├── notebooks/               # обучение LightGBM (uv, Python)
│   ├── reports/                 # отчёты AUC, калибровка, SHAP
│   ├── features.json            # манифест фич (общий с Go-экстрактором)
│   └── golden/                  # golden-строки для Go-теста инференса
├── loadtest/k6/                 # нагрузочный сценарий
├── .github/workflows/
│   ├── ci.yml                   # обязательные проверки на PR
│   └── deploy.yml               # тесты → E2E → build/push → deploy
├── docker-compose.yml           # локальная разработка
├── docker-compose.e2e.yml       # лёгкий стек для E2E/K6
├── docker-compose.prod.yml      # продакшен
├── Caddyfile
├── CONTRIBUTING.md
└── Makefile
```
---

## Backend

Слоистая архитектура, обработка ошибок, инструкция по добавлению ручек, миграции и раскладка тестов — в [CONTRIBUTING.md](CONTRIBUTING.md).

### Embeddings (TEI)

Вектора генерирует [Text Embeddings Inference](https://github.com/huggingface/text-embeddings-inference) с моделью `intfloat/multilingual-e5-small`. В docker-compose он поднимается как сервис `tei` и доступен по внутреннему адресу `http://tei:80` (`TEI_URL`), наружу проброшен на `TEI_EXTERNAL_PORT` (по умолчанию `8090`).

Провайдер выбирается конфигом `EMBEDDING_PROVIDER`: `tei` — реальный сервис, либо `stub` (по умолчанию) — детерминированный клиент-заглушка для разработки без TEI. Остальные настройки: `VECTOR_DIM`, `EMBEDDING_TIMEOUT`, `MAX_INPUT_LENGTH` — лимит входного текста в символах (текст усекается до него перед отправкой в модель).

Проверить, что TEI поднялся и отвечает:

```bash
docker compose up -d tei
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8090/health   # ждём 200
curl -s -X POST http://localhost:8090/embed -H "Content-Type: application/json" -d '{"inputs":"хочу iPhone"}'
```

Если `EMBEDDING_PROVIDER=tei`, `app` стартует только после того, как `tei` стал healthy (внутренний healthcheck ходит на `/health`).

### Справочник API

Swagger UI отдаётся backend'ом по адресу [`/swagger/index.html`](http://localhost:8080/swagger/index.html), OpenAPI-спек — на `/swagger/doc.json`. Ручки размечены аннотациями [swaggo/swag](https://github.com/swaggo/swag); после изменения ручек или DTO перегенерируйте спек:

```bash
# один раз: go install github.com/swaggo/swag/cmd/swag@latest
cd backend && swag init -g cmd/api/main.go -o docs --parseInternal
```

См. также: [CONTRIBUTING.md](CONTRIBUTING.md) · [Swagger UI](http://localhost:8080/swagger/index.html).

---

## Frontend

### Окружение

| Переменная | Значение по умолчанию | Назначение |
|------------|-----------------------|------------|
| `VITE_API_BASE_URL` | (пусто, relative `/api`) | Базовый URL API (префикс `/api/v1` добавляется автоматически). Пусто = через Vite-прокси на `:8080` |

Запрос с пустым `VITE_API_BASE_URL` идёт relative через Vite-прокси (`vite.config.ts`, `server.proxy`) на `http://localhost:8080` — бэкенду не нужен CORS. Фото приходят как `imageUrl` из `MINIO_PUBLIC_ENDPOINT` и используются как есть.

### Feature-Sliced Design

Слои (сверху вниз, импорты только вниз):

- `src/app` — провайдеры, роутинг, глобальные стили, стор.
- `src/pages` — экраны.
- `src/features` — пользовательские сценарии.
- `src/entities` — бизнес-сущности.
- `src/shared` — переиспользуемые UI-компоненты, API-клиент, утилиты, типы.

### Скрипты

```bash
npm run lint          # ESLint (включая правила импортов FSD)
npm run lint:fix
npm run format        # Prettier
npm run format:check
npm run test          # Vitest
npm run typecheck     # tsc --noEmit
```

---

## Команда

Проект разработан в рамках хакатона Авито «Лаборатория Кода».

Бэкенд: [aabbuukkaarr8](https://github.com/aabbuukkaarr8), [Linempy](https://github.com/Linempy), [Danya-programmer](https://github.com/Danya-programmer).