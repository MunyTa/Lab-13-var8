# Агенты HR Multi-Agent системы

## Обзор агентов

Система включает 4 агента, каждый из которых выполняет специализированную роль в HR-воронке найма.

---

## 1. Resume Parser Agent

### Роль
Парсинг и нормализация резюме кандидатов.

### Входные данные
```json
{
  "id": "task-uuid",
  "type": "resume_parse",
  "payload": {
    "raw_text": "Имя: Иван Петров\nEmail: ivan@example.com\nНавыки: Go, Python..."
  }
}
```

### Выходные данные
```json
{
  "task_id": "task-uuid",
  "success": true,
  "resume": {
    "id": "task-uuid",
    "email": "ivan@example.com",
    "phone": "+7-999-123-4567",
    "skills": ["Go", "Python", "Docker"],
    "experience": ["Senior Developer в TechCorp"],
    "education": ["МГУ, Прикладная математика"]
  },
  "parsed": {
    "Name": "Иван Петров",
    "Email": "ivan@example.com",
    "Skills": ["Go", "Python", "Docker"],
    "Experience": ["Senior Developer..."],
    "Education": ["МГУ"]
  }
}
```

### Бизнес-правила
- Извлекает email с помощью регулярных выражений
- Телефон парсится из различных форматов
- Навыки классифицируются по ключевым словам
- Опыт работы фильтруется по ключевым словам

---

## 2. Vacancy Matcher Agent

### Роль
Сопоставление кандидатов с вакансиями на основе навыков.

### Входные данные
- Результат работы Resume Parser Agent

### Выходные данные
```json
{
  "task_id": "task-uuid",
  "success": true,
  "match_score": 75.0,
  "candidate": {
    "resume": { ... },
    "vacancy": {
      "id": "vac-hr-default",
      "title": "Go Backend Developer",
      "skills": ["Go", "Docker", "Kubernetes", "PostgreSQL"]
    },
    "match_score": 75.0,
    "strengths": ["Go", "Docker", "PostgreSQL"],
    "weaknesses": ["Kubernetes"]
  }
}
```

### Бизнес-правила
- Match score = (совпадающие навыки / требуемые навыки) * 100
- Если score >= 70% - кандидат проходит дальше
- Strengths: навыки кандидата, совпадающие с требованиями
- Weaknesses: требуемые навыки, которых нет у кандидата

---

## 3. Interview Scheduler Agent

### Роль
Планирование собеседований с подбором удобного времени.

### Входные данные
- Результат работы Vacancy Matcher Agent
- Предпочтения по дате/времени

### Выходные данные
```json
{
  "task_id": "task-uuid",
  "success": true,
  "interview": {
    "id": "int-abc123",
    "candidate_id": "candidate-uuid",
    "candidate_name": "Иван Петров",
    "position": "Go Backend Developer",
    "scheduled_at": "2024-01-22T10:00:00Z",
    "duration_minutes": 60,
    "location": "Conference Room A / Zoom",
    "interviewers": ["HR Manager", "Tech Lead"],
    "status": "scheduled"
  }
}
```

### Бизнес-правила
- Генерирует слоты с 9:00 до 18:00 с шагом 60 минут
- Выбирает первый доступный слот
- Назначает 2 интервьюера по умолчанию
- Срок проведения: через 7 дней после подачи

---

## 4. Feedback Agent

### Роль
Сбор и анализ обратной связи после собеседований.

### Входные данные
- Результат работы Interview Scheduler Agent

### Выходные данные
```json
{
  "task_id": "task-uuid",
  "success": true,
  "avg_score": 8.5,
  "output": "Interview Summary Report...",
  "feedbacks": [
    {
      "id": "fb-int-abc123-1",
      "rating": 8,
      "pros": ["Strong technical skills", "Good communication"],
      "cons": ["Could improve problem-solving speed"],
      "recommendation": "Hire"
    }
  ]
}
```

### Бизнес-правила
- Генерирует mock-feedback от каждого интервьюера
- Average score = среднее от всех оценок
- Рекомендации:
  - >= 9: Strong Hire
  - >= 7: Hire
  - >= 5: No Hire
  - < 5: Strong No Hire
- Сохраняет результаты в Redis

---

## Схема коммуникации

```
[Клиент] 
    │
    ▼
POST /pipeline/hr ───────────────────────────────────┐
    │                                              │
    ▼                                              │
[Resume Parser] ──────► [Vacancy Matcher] ──────────►│
hr.resume.parse      hr.vacancy.match               │
                                                 [▼]
                                            [Interview Scheduler]
                                            hr.interview.schedule
                                                 [▼]
                                            [Feedback Agent]
                                            hr.feedback.collect
                                                 │
                                                 ▼
                                        [hr.tasks.completed]
                                                 │
                                                 ▼
                                          [Оркестратор]
                                        ┌─────────┴─────────┐
                                        │                 │
                                        ▼                 ▼
                                    [API Response]    [Redis]
```

---

## Метрики и мониторинг

### Счётчики
- `hr:processed_tasks` - общее количество обработанных задач
- `hr:total_candidates_processed` - общее количество кандидатов
- `hr:candidate:{id}` - хеш с данными кандидата (score, recommendation)

### Логирование
- Каждый агент логирует при ENABLE_LOGGING=true
- Уровни: INFO, ERROR
- Вывод: stdout + файл

### Трассировка
- OpenTelemetry интегрирован во все агенты
- Экспорт в Jaeger на jaeger:4317
