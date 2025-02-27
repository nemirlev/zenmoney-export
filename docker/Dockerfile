FROM golang:1.24-alpine AS builder

WORKDIR /build

# Копируем только go.mod и go.sum для лучшего кэширования слоёв
COPY ./go.mod ./go.sum ./
RUN go mod download

# Копируем исходники
COPY . .

# Собираем статически скомпилированный бинарник
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o zenexport main.go

# Конечный образ намного меньше, используем scratch
FROM alpine:latest

# Добавляем ca-certificates для HTTPS и создаем непривилегированного пользователя
RUN apk --no-cache add ca-certificates && \
    addgroup -S appgroup && adduser -S appuser -G appgroup

# Копируем бинарник из предыдущего этапа
COPY --from=builder /build/zenexport /app/zenexport

# Используем непривилегированного пользователя
USER appuser

WORKDIR /app

# Документируем порты
EXPOSE 8080

# Определяем точку входа
ENTRYPOINT ["/app/zenexport", "sync", "-d"]