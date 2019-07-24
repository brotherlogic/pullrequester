package main

import (
	"fmt"

	"golang.org/x/net/context"

	pb "github.com/brotherlogic/pullrequester/proto"
)

func (s *Server) update(ctx context.Context, req, reqIn *pb.PullRequest) (*pb.UpdateResponse, error) {
	if reqIn.NumberOfCommits > 0 {
		req.NumberOfCommits = reqIn.NumberOfCommits
	}

	for _, check := range reqIn.Checks {
		found := false
		for _, checkIn := range req.Checks {
			if checkIn.Source == check.Source {
				found = true
				checkIn.Pass = check.Pass
			}
		}

		if !found {
			req.Checks = append(req.Checks, check)
		}
	}

	for _, sha := range reqIn.Shas {
		found := false
		for _, cSha := range req.Shas {
			if cSha == sha {
				found = true
			}
		}

		if !found {
			req.Shas = append(req.Shas, sha)
		}
	}

	return &pb.UpdateResponse{}, nil
}

// UpdatePullRequest updates the pull request
func (s *Server) UpdatePullRequest(ctx context.Context, req *pb.UpdateRequest) (*pb.UpdateResponse, error) {
	if len(req.Update.Url) > 0 {
		for _, pr := range s.config.Tracking {
			if pr.Url == req.Update.Url {
				return s.update(ctx, pr, req.Update)
			}
		}
		s.config.Tracking = append(s.config.Tracking, req.Update)
		return &pb.UpdateResponse{}, nil
	}

	if len(req.Update.Shas) > 0 {
		for _, pr := range s.config.Tracking {
			for _, sha := range pr.Shas {
				if sha == req.Update.Shas[0] {
					return s.update(ctx, pr, req.Update)
				}
			}
		}
	}

	return nil, fmt.Errorf("Unable to locate PR %v", req)
}
