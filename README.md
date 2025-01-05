# grpc-microservices
Идея взаимодействия сервисов:
TaskService: Учет задач (Task Manager API). Позволяет добавлять, получать и обновлять задачи.
NotificationService: Уведомления (Notification API). Получает информацию о новых задачах от TaskService и сохраняет её в своей базе.