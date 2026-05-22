# PROMPT_LOG.md - Журнал создания лабораторной работы

## Этап 1: Анализ методички

### Вариант 8: Автоматизация HR

**Типы агентов:**
1. Парсинг резюме
2. Сопоставление с вакансиями
3. Планирование собеседований
4. Обратная связь

**Сложность:** Средний уровень → выполнено как повышенный (вариант 8, но с полной реализацией как в вариантах 11-30)

---

## Этап 2: Изучение референса

Изучена структура репозитория https://github.com/ViTaMiR-4iK/LR-13 (вариант 17, Кибербезопасность SIEM)

### Структура, взятая за основу:
- 4 Go-агента (log-collector, event-correlator, attack-detector, traffic-blocker)
- Python/FastAPI оркестратор
- docker-compose.yml с NATS, Redis, Jaeger
- OpenTelemetry трассировка
- Веб-интерфейс мониторинга
- LLM-агент для Ollama

---

## Этап 3: Адаптация под вариант 8

### Адаптированные компоненты:

| Оригинал (SIEM) | Адаптация (HR) |
|-----------------|----------------|
| log-collector | resume-parser |
| event-correlator | vacancy-matcher |
| attack-detector | interview-scheduler |
| traffic-blocker | feedback-agent |

### Subject изменения:
- siem.logs.collect → hr.resume.parse
- siem.events.correlate → hr.vacancy.match
- siem.attacks.detect → hr.interview.schedule
- siem.traffic.block → hr.feedback.collect

---

## Этап 4: Реализация

### Созданные файлы:

#### Go модуль
- `go.mod` - модуль с зависимостями
- `internal/hr/types.go` - общие типы данных
- `internal/hr/parser.go` - логика парсинга резюме
- `internal/hr/matcher.go` - логика сопоставления
- `internal/hr/scheduler.go` - логика планирования
- `internal/hr/feedback.go` - логика обработки обратной связи
- `internal/hr/hr_test.go` - unit-тесты

#### Go агенты
- `agents/parser/main.go` + `Dockerfile.parser`
- `agents/matcher/main.go` + `Dockerfile.matcher`
- `agents/scheduler/main.go` + `Dockerfile.scheduler`
- `agents/feedback/main.go` + `Dockerfile.feedback`

#### Python компоненты
- `orchestrator/main.py` - FastAPI приложение
- `orchestrator/llm_agent.py` - LLM агент
- `tests/test_orchestrator.py` - pytest тесты

#### Инфраструктура
- `docker-compose.yml` - NATS, Redis, Jaeger, агенты
- `Dockerfile.python` - для Python сервисов
- `requirements.txt` - Python зависимости
- `pytest.ini` - pytest конфигурация

#### Документация и скрипты
- `README.md` - основная документация
- `docs/agent_roles.md` - подробное описание ролей
- `scripts/run_demo.ps1` - демо скрипт
- `scripts/scale_agents.py` - скрипт масштабирования
- `web/templates/index.html` - веб-интерфейс
- `.gitignore` - git ignore файл

---

## Этап 5: Особенности реализации

### HR-специфичная логика:
1. **Resume Parser** - извлекает email, телефон, навыки, опыт, образование из текста резюме
2. **Vacancy Matcher** - вычисляет match score на основе совпадающих навыков
3. **Interview Scheduler** - генерирует временные слоты, назначает интервьюеров
4. **Feedback Agent** - собирает mock-feedback, вычисляет avg score, формирует рекомендации

### Рекомендации по найму:
- >= 9 баллов: Strong Hire
- >= 7 баллов: Hire
- >= 5 баллов: No Hire
- < 5 баллов: Strong No Hire

---

## Этап 6: Интеграции

### OpenTelemetry:
- Трассировка добавлена во все Go агенты
- Экспорт в Jaeger на jaeger:4317
- Trace ID и Span ID передаются через NATS headers

### Redis:
- feedback-agent сохраняет данные кандидатов
- Счётчики processed_tasks, total_candidates_processed

### LLM Agent (Ollama):
- Опциональная интеграция при наличии OLLAMA_URL
- Генерация объяснений и интервью-вопросов

---

## Команда запуска

```bash
cd d:/projects/Новая папка/LR-13
docker compose up --build
```

---

## Версия

Создано: 22 мая 2026
Автор: Кузьмищев Родион Ильич
Группа: 221331
Вариант: 8
