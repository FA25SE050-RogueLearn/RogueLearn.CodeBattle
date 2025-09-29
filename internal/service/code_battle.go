package service

import (
	"context"
	"log/slog"

	pb "github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/api"
	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/internal/store"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type CodeBattleServer struct {
	pb.UnimplementedCodeBattleServiceServer
	queries *store.Queries
	logger  *slog.Logger
}

func NewCodeBattleServer(queries *store.Queries, logger *slog.Logger) *CodeBattleServer {
	return &CodeBattleServer{
		queries: queries,
		logger:  logger,
	}
}

func (s *CodeBattleServer) GetEvents(ctx context.Context, req *pb.GetEventsRequest) (*pb.GetEventsResponse, error) {
	events, err := s.queries.GetEvents(ctx)
	if err != nil {
		s.logger.Error("err at getting events", "err", err)
		status := pb.Status{Success: false, Message: "get events failed", ErrorMessage: err.Error()}
		return &pb.GetEventsResponse{
			Status: &status,
			Events: nil,
		}, err
	}

	s.logger.Info("successfully getting events", "events", events)
	pbEvents := convertStoreEventsToPB(events)
	resp := pb.GetEventsResponse{
		Status: &pb.Status{
			Success:      true,
			Message:      "successfully getting events",
			ErrorMessage: "",
		},
		Events: pbEvents,
	}

	s.logger.Info("successfully converting to pbevents", "pbevents", pbEvents)
	return &resp, nil
}

func convertStoreEventsToPB(storeEvents []store.Event) []*pb.Event {
	pbEvents := make([]*pb.Event, len(storeEvents))
	for i, e := range storeEvents {
		pbEvents[i] = &pb.Event{
			Id:          e.ID.String(),
			Title:       e.Title,
			Description: e.Description,
			// change later
			Type:      pb.EventType_EVENT_TYPE_CODE_BATTLE,
			StartDate: timestamppb.New(e.StartedDate.Time),
			EndDate:   timestamppb.New(e.EndDate.Time),
		}
	}
	return pbEvents
}
