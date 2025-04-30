# 🚀 Highload HTTP Load Balancer

![Go](https://img.shields.io/badge/Go-1.19+-00ADD8?logo=go)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-13+-336791?logo=postgresql)
![Docker](https://img.shields.io/badge/Docker-20.10+-2496ED?logo=docker)

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
- API для управления лимитами

### 🗄 Хранение данных
- PostgreSQL для хранения клиентов

## 🛠 Быстрый старт

### Требования
-Docker 20.10+
-Docker Compose 2.0+
```bash
git clone https://github.com/your-repo/highload-balancer.git
cd highload-balancer
docker-compose up --build
```
### 📚 Документация API

