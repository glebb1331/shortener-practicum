# go-musthave-shortener-tpl

Шаблон репозитория для трека «Сервис сокращения URL».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m v2 template https://github.com/Yandex-Practicum/go-musthave-shortener-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/v2 .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Структура проекта

Приведённая в этом репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:
- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**

## Профилирование и бенчмарки

Бенчмарки:

```bash
go test -bench=. -benchmem ./internal/handler/ ./internal/storage/
```

Дифф профилей:

```bash
go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof
```

```
      flat  flat%   sum%        cum   cum%
-1080.27MB 25.66% 25.66% -1080.27MB 25.66%  storage.(*MemoryStorage).GetByUserID
   10.50MB  0.25% 25.41%       10MB  0.24%  fmt.Sprintf
       4MB 0.095% 25.31%    12.98MB  0.31%  storage.BenchmarkGet
         0     0% 25.31% -1081.35MB 25.68%  storage.BenchmarkGetByUserID
```

Что поменяли:
- `generateID` — `sync.Pool` вместо `make` каждый раз
- `urlPrefix` — строковая конкатенация вместо `url.JoinPath`
- `ShortenBatch` — один мьютекс на весь цикл
- `GetByUserID` — `make([]Record, 0, len/2)` вместо `nil`

Покрытие тестами: **42.5%**
