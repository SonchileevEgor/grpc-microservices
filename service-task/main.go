package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	pb "github.com/SonchileevEgor/grpc-microservices/service-notification/notifications"
	"github.com/jackc/pgx/v4/pgxpool"
	"google.golang.org/grpc"
)

// Task представляет задачу
type Task struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

// NotificationClient обертка для gRPC клиента
type NotificationClient struct {
	client pb.NotificationServiceClient
}

// NewNotificationClient создает новый клиент NotificationService
func NewNotificationClient(address string) *NotificationClient {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Не удалось подключиться к gRPC серверу: %v", err)
	}
	client := pb.NewNotificationServiceClient(conn)
	return &NotificationClient{client: client}
}

// Вспомогательная функция для отправки уведомления
func sendNotification(client *NotificationClient, taskID int, message string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.client.SaveNotification(ctx, &pb.SaveNotificationRequest{
		TaskId:  int32(taskID),
		Message: message,
	})
	if err != nil {
		log.Printf("Ошибка при отправке уведомления: %v", err)
	}
}

// Обработчик создания задачи
func createTaskHandler(db *pgxpool.Pool, notificationClient *NotificationClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
			return
		}

		var task Task
		if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
			http.Error(w, "Некорректный формат данных", http.StatusBadRequest)
			return
		}

		query := `INSERT INTO tasks (title, description, completed) VALUES ($1, $2, $3) RETURNING id`
		err := db.QueryRow(context.Background(), query, task.Title, task.Description, task.Completed).Scan(&task.ID)
		if err != nil {
			http.Error(w, "Ошибка при создании задачи", http.StatusInternalServerError)
			return
		}

		// Отправка уведомления
		go sendNotification(notificationClient, task.ID, "Создана новая задача: "+task.Title)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(task)
	}
}

// Обработчик получения задачи
func getTaskHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
			return
		}

		idStr := r.URL.Query().Get("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Некорректный ID", http.StatusBadRequest)
			return
		}

		var task Task
		query := `SELECT id, title, description, completed FROM tasks WHERE id = $1`
		err = db.QueryRow(context.Background(), query, id).Scan(&task.ID, &task.Title, &task.Description, &task.Completed)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Задача не найдена", http.StatusNotFound)
			} else {
				http.Error(w, "Ошибка при получении задачи", http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(task)
	}
}

// Обработчик обновления задачи
func updateTaskHandler(db *pgxpool.Pool, notificationClient *NotificationClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
			return
		}

		var task Task
		if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
			http.Error(w, "Некорректный формат данных", http.StatusBadRequest)
			return
		}

		query := `UPDATE tasks SET title = $1, description = $2, completed = $3 WHERE id = $4`
		_, err := db.Exec(context.Background(), query, task.Title, task.Description, task.Completed, task.ID)
		if err != nil {
			http.Error(w, "Ошибка при обновлении задачи", http.StatusInternalServerError)
			return
		}

		// Отправка уведомления
		go sendNotification(notificationClient, task.ID, "Обновлена задача: "+task.Title)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(task)
	}
}

// Обработчик удаления задачи
func deleteTaskHandler(db *pgxpool.Pool, notificationClient *NotificationClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
			return
		}

		idStr := r.URL.Query().Get("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Некорректный ID", http.StatusBadRequest)
			return
		}

		query := `DELETE FROM tasks WHERE id = $1`
		_, err = db.Exec(context.Background(), query, id)
		if err != nil {
			http.Error(w, "Ошибка при удалении задачи", http.StatusInternalServerError)
			return
		}

		// Отправка уведомления
		go sendNotification(notificationClient, id, "Удалена задача с ID: "+strconv.Itoa(id))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "Задача успешно удалена"})
	}
}

func main() {
	// Подключение к базе данных
	dbURL := os.Getenv("DB_URL")
    db, err := pgxpool.Connect(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	defer db.Close()

	// Инициализация gRPC клиента
	notificationClient := NewNotificationClient("service-notification:50052")

	// Роуты
	http.HandleFunc("/tasks", createTaskHandler(db, notificationClient))
	http.HandleFunc("/tasks/get", getTaskHandler(db))
	http.HandleFunc("/tasks/update", updateTaskHandler(db, notificationClient))
	http.HandleFunc("/tasks/delete", deleteTaskHandler(db, notificationClient))

	// Запуск сервера
	fmt.Println("HTTP сервер запущен на :8081")
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatalf("Ошибка запуска HTTP сервера: %v", err)
	}
}
