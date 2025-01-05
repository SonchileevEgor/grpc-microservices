package main

import (
    "context"
    "log"
    "net"
	"time"
	"os"

    "github.com/jackc/pgx/v4/pgxpool"
    pb "service-task/tasks"
    "google.golang.org/grpc"
)

type server struct {
    pb.UnimplementedTaskServiceServer
    db *pgxpool.Pool
}

// notifyServiceClient создаёт gRPC клиент для Service B.
func notifyServiceClient(ctx context.Context, address string) (pb.NotificationServiceClient, *grpc.ClientConn, error) {
	conn, err := grpc.DialContext(ctx, address, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(5*time.Second))
	if err != nil {
		return nil, nil, err
	}
	return pb.NewNotificationServiceClient(conn), conn, nil
}

func (s *server) CreateTask(ctx context.Context, req *pb.CreateTaskRequest) (*pb.CreateTaskResponse, error) {
	var id int32
	err := s.db.QueryRow(ctx, "INSERT INTO tasks (title, description) VALUES ($1, $2) RETURNING id", req.Title, req.Description).Scan(&id)
	if err != nil {
		return nil, err
	}

	// Вызов Service B.
	notificationServiceURL := os.Getenv("NOTIFICATION_SERVICE_URL")
	notificationClient, conn, err := notifyServiceClient(ctx, notificationServiceURL)
	if err != nil {
		log.Printf("failed to connect to NotificationService: %v", err)
		return &pb.CreateTaskResponse{Id: id}, nil // Продолжаем без уведомлений.
	}
	defer conn.Close()

	_, err = notificationClient.SaveNotification(ctx, &pb.SaveNotificationRequest{
		TaskId:  id,
		Message: "New task created: " + req.Title,
	})
	if err != nil {
		log.Printf("failed to notify Service B: %v", err)
	}
	return &pb.CreateTaskResponse{Id: id}, nil
}

func main() {
	dbURL := os.Getenv("DB_URL")
    db, err := pgxpool.Connect(context.Background(), dbURL)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    lis, err := net.Listen("tcp", ":50051")
    if err != nil {
        log.Fatal(err)
    }

    grpcServer := grpc.NewServer()
    pb.RegisterTaskServiceServer(grpcServer, &server{db: db})
    log.Println("Service A running on :50051")
    grpcServer.Serve(lis)
}
