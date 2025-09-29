package main

import (
	"context"
	"log"
	"time"

	pb "github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Set up a connection to the server.
	conn, err := grpc.NewClient("localhost:8080", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewCodeBattleServiceClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &pb.GetEventsRequest{
		Pagination: &pb.Pagination{
			PageSize:  10,
			PageIndex: 1,
		},
	}

	r, err := c.GetEvents(ctx, req)
	if err != nil {
		log.Fatalf("could not get events: %v", err)
	}

	log.Printf("Events received:")
	for _, event := range r.GetEvents() {
		log.Printf("  ID: %s, Title: %s, Description: %s, Type: %s",
			event.GetId(), event.GetTitle(), event.GetDescription(), event.GetType().String())
		if event.GetStartDate() != nil {
			log.Printf("    Start Date: %s", event.GetStartDate().AsTime().Format(time.RFC3339))
		}
		if event.GetEndDate() != nil {
			log.Printf("    End Date: %s", event.GetEndDate().AsTime().Format(time.RFC3339))
		}

		log.Println("status:", r.Status)
	}
}
