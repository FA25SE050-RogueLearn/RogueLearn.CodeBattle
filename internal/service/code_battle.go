package service

import (
	"context"
	"log/slog"

	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/internal/store"
	pb "github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/protos"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	DefaultGetLimit  = 10
	DefaultGetOffset = 0
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
	events, err := s.queries.GetEvents(ctx, store.GetEventsParams{Limit: DefaultGetLimit, Offset: DefaultGetOffset})
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

func (s *CodeBattleServer) SubmitCodeSolution(ctx context.Context, req *pb.SubmitCodeSolutionRequest) (*pb.SubmitCodeSolutionResponse, error) {
	cpid, err := uuid.Parse(req.CodeProblemId)
	if err != nil {
		s.logger.Error("err at parsing code problem id", "err", err)
		status := pb.Status{Success: false, Message: "parse code problem id failed", ErrorMessage: err.Error()}
		return &pb.SubmitCodeSolutionResponse{
			Status: &status,
		}, err
	}

	uid, err := uuid.Parse(req.UserId)
	if err != nil {
		s.logger.Error("err at parsing user id", "err", err)
		status := pb.Status{Success: false, Message: "parse user id failed", ErrorMessage: err.Error()}
		return &pb.SubmitCodeSolutionResponse{
			Status: &status,
		}, err
	}

	lid, err := uuid.Parse(req.LanguageId)
	if err != nil {
		s.logger.Error("err at parsing language id", "err", err)
		status := pb.Status{Success: false, Message: "parse language id failed", ErrorMessage: err.Error()}
		return &pb.SubmitCodeSolutionResponse{
			Status: &status,
		}, err
	}

	rid, err := uuid.Parse(req.RoomId)
	if err != nil {
		s.logger.Error("err at parsing room id", "err", err)
		status := pb.Status{Success: false, Message: "parse room id failed", ErrorMessage: err.Error()}
		return &pb.SubmitCodeSolutionResponse{
			Status: &status,
		}, err
	}

	// TODO: get guild_id

	submission, err := s.queries.CreateSubmission(ctx, store.CreateSubmissionParams{
		UserID: pgtype.UUID{
			Bytes: uid,
			Valid: true,
		},
		LanguageID: pgtype.UUID{
			Bytes: lid,
			Valid: true,
		},
		RoomID: pgtype.UUID{
			Bytes: rid,
			Valid: true,
		},
		CodeProblemID: pgtype.UUID{
			Bytes: cpid,
			Valid: true,
		},
		CodeSubmitted: req.CodeSubmitted,
		Status:        store.SubmissionStatusPending,
	})
	if err != nil {
		return &pb.SubmitCodeSolutionResponse{
			Status: &pb.Status{
				Success:      false,
				Message:      "failed to submit code solution",
				ErrorMessage: err.Error(),
			},
			Submission: submissionToPB(&submission),
		}, err
	}

	return nil, nil
}

func convertStoreEventsToPB(storeEvents []store.Event) []*pb.Event {
	pbEvents := make([]*pb.Event, len(storeEvents))
	for i, e := range storeEvents {
		pbEvents[i] = &pb.Event{
			Id:          e.ID.String(),
			Title:       e.Title,
			Description: e.Description,
			// change later
			Type:      pb.EventType_code_battle,
			StartDate: timestamppb.New(e.StartedDate.Time),
			EndDate:   timestamppb.New(e.EndDate.Time),
		}
	}
	return pbEvents
}

func submissionToPB(s *store.Submission) *pb.Submission {
	return &pb.Submission{
		Id:            s.ID.String(),
		UserId:        s.UserID.String(),
		LanguageId:    s.LanguageID.String(),
		RoomId:        s.RoomID.String(),
		CodeProblemId: s.CodeProblemID.String(),
		CodeSubmitted: s.CodeSubmitted,
		Status:        pb.SubmissionStatus(pb.SubmissionStatus_value[string(s.Status)]),
	}
}
