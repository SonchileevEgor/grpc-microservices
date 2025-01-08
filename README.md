# grpc-microservices
Идея взаимодействия сервисов:<br />
TaskService: Учет задач (Task Manager API). Позволяет добавлять, получать и обновлять задачи.<br />
NotificationService: Уведомления (Notification API). Получает информацию о новых задачах от TaskService и сохраняет её в своей базе.<br />

migrations:<br />
goose -dir ./migrations postgres "host=localhost port=5433 user=user_notification password=password_notification dbname=db_notification sslmode=disable" up<br />
goose -dir ./migrations postgres "host=localhost port=5432 user=user_task password=password_task dbname=db_task sslmode=disable" up<br />