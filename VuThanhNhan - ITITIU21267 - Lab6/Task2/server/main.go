package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"net"

	pb "book-catalog-grpc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type calculatorServer struct {
	pb.UnimplementedCalculatorServer
	history []string
}

func (s *calculatorServer) Calculate(ctx context.Context, req *pb.CalculateRequest) (*pb.CalculateResponse, error) {
	log.Printf("Calculate: %.2f %s %.2f", req.A, req.Operation, req.B)

	var result float32

	switch req.Operation {
	case "add":
		result = req.A + req.B
	case "subtract":
		result = req.A - req.B
	case "multiply":
		result = req.A * req.B
	case "divide":
		if req.B == 0 {
			return nil, status.Errorf(codes.InvalidArgument, "cannot divide by zero")
		}
		result = req.A / req.B
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unknown operation: %s", req.Operation)
	}

	entry := fmt.Sprintf("%.2f %s %.2f = %.2f", req.A, req.Operation, req.B, result)
	s.history = append(s.history, entry)

	return &pb.CalculateResponse{
		Result:    result,
		Operation: req.Operation,
	}, nil
}

func (s *calculatorServer) SquareRoot(ctx context.Context, req *pb.SquareRootRequest) (*pb.SquareRootResponse, error) {
	log.Printf("SquareRoot: %.2f", req.Number)

	if req.Number < 0 {
		return nil, status.Errorf(codes.InvalidArgument,
			"cannot calculate square root of negative number: %.2f", req.Number)
	}

	result := float32(math.Sqrt(float64(req.Number)))

	entry := fmt.Sprintf("sqrt(%.2f) = %.2f", req.Number, result)
	s.history = append(s.history, entry)

	return &pb.SquareRootResponse{
		Result: result,
	}, nil
}

func (s *calculatorServer) GetHistory(ctx context.Context, req *pb.HistoryRequest) (*pb.HistoryResponse, error) {
	log.Println("GetHistory called")
	return &pb.HistoryResponse{
		Calculations: s.history,
		Count:        int32(len(s.history)),
	}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterCalculatorServer(grpcServer, &calculatorServer{})

	log.Println("🚀 Calculator gRPC server listening on :50051")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
