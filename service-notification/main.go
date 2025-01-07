package main

import (
    "context"
    "log"
    "net"
	"os"

    "github.com/jackc/pgx/v4/pgxpool"
    pb "github.com/SonchileevEgor/grpc-microservices/service-notification/notifications"
    "google.golang.org/grpc"
)

type server struct {
    pb.UnimplementedNotificationServiceServer
    db *pgxpool.Pool
}

func (s *server) SaveNotification(ctx context.Context, req *pb.SaveNotificationRequest) (*pb.SaveNotificationResponse, error) {
    _, err := s.db.Exec(ctx, "INSERT INTO notifications (task_id, message) VALUES ($1, $2)", req.TaskId, req.Message)
    if err != nil {
        return &pb.SaveNotificationResponse{Success: false}, err
    }
    return &pb.SaveNotificationResponse{Success: true}, nil
}

func main() {
	dbURL := os.Getenv("DB_URL")
    db, err := pgxpool.Connect(context.Background(), dbURL)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    lis, err := net.Listen("tcp", ":50052")
    if err != nil {
        log.Fatal(err)
    }

    grpcServer := grpc.NewServer()
    pb.RegisterNotificationServiceServer(grpcServer, &server{db: db})
    log.Println("Service B running on :50052")
    grpcServer.Serve(lis)
}
