package main

import "golang.org/x/net/context"

import pb "github.com/brotherlogic/pullrequester/proto"

func (s *Server) updateChecks(check *pb.PullRequest_Check, req *pb.PullRequest) {
	if check.Pass != pb.PullRequest_Check_PASS {
		for _, checkFail := range req.Checks {
			checkFail.Pass = pb.PullRequest_Check_FAIL
		}
	}
}

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
				s.updateChecks(check, req)
			}
		}

		if !found {
			req.Checks = append(req.Checks, check)
			s.updateChecks(check, req)
		}
	}

	return &pb.UpdateResponse{}, nil
}

// UpdatePullRequest updates the pull request
func (s *Server) UpdatePullRequest(ctx context.Context, req *pb.UpdateRequest) (*pb.UpdateResponse, error) {
	for _, pr := range s.config.Tracking {
		if pr.Url == req.Update.Url {
			return s.update(ctx, pr, req.Update)
		}
	}

	s.config.Tracking = append(s.config.Tracking, req.Update)
	return &pb.UpdateResponse{}, nil
}
