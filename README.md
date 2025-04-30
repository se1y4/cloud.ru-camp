# 🚀 Highload HTTP Load Balancer

![Go](https://img.shields.io/badge/Go-1.19+-00ADD8?logo=go)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-13+-336791?logo=postgresql)
![Docker](https://img.shields.io/badge/Docker-20.10+-2496ED?logo=docker)
![GitHub Last Commit](https://img.shields.io/github/last-commit/your-repo/highload-balancer)

Профессиональный балансировщик нагрузки с поддержкой rate-limiting и управлением клиентами через REST API.

## 📦 Основные возможности

### 🔄 Балансировка нагрузки
- Поддержка алгоритмов:
  - Round Robin
  - Least Connections
- Автоматические health checks бэкендов
- Конфигурация через YAML-файл

### ⏱ Rate Limiting
- Алгоритм Token Bucket
- Индивидуальные лимиты для клиентов
- REST API для управления лимитами

### 🗄 Хранение данных
- PostgreSQL для хранения клиентов
- Автоматические миграции
- Резервное копирование

## 🛠 Быстрый старт

### Требования
```bash
Docker 20.10+
Docker Compose 2.0+
