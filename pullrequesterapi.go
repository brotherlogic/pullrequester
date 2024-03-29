package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/context"

	pbgh "github.com/brotherlogic/githubcard/proto"
	pb "github.com/brotherlogic/pullrequester/proto"
)

func (s *Server) updatePR(ctx context.Context, req *pb.PullRequest) {
	if len(req.Url) > 0 {
		elems := strings.Split(req.Url, "/")
		val, _ := strconv.Atoi(elems[7])
		prs, err := s.github.getPullRequest(ctx, &pbgh.PullRequest{Job: elems[5], PullNumber: int32(val)})
		if err == nil {
			req.NumberOfCommits = prs.NumberOfCommits
		}
	}
}

func (s *Server) updateChecks(check *pb.PullRequest_Check, req *pb.PullRequest) {
	if check.Pass != pb.PullRequest_Check_PASS {
		for _, checkFail := range req.Checks {
			checkFail.Pass = pb.PullRequest_Check_FAIL
		}
	}
}

func (s *Server) processPullRequest(ctx context.Context, pr *pb.PullRequest) error {
	if pr.NumberOfCommits == 1 && len(pr.Checks) == 4 {
		for _, check := range pr.Checks {
			if check.Pass != pb.PullRequest_Check_PASS {
				return fmt.Errorf("PR is not passing tests")
			}
		}

		s.CtxLog(ctx, fmt.Sprintf("Ready for Auto Merge %v", pr.Url))
		elems := strings.Split(pr.Url, "/")
		val, _ := strconv.Atoi(elems[7])

		resp, err := s.github.closePullRequest(ctx, &pbgh.CloseRequest{Job: elems[5], PullNumber: int32(val), Sha: pr.Shas[len(pr.Shas)-1], BranchName: pr.Name})
		s.CtxLog(ctx, fmt.Sprintf("Result %v, %v", resp, err))
		return err
	}

	return fmt.Errorf("PR is not ready for auto merge: %v", pr)
}

func (s *Server) update(ctx context.Context, req, reqIn *pb.PullRequest) (*pb.UpdateResponse, error) {
	defer s.updatePR(ctx, req)

	if len(reqIn.Name) > 0 {
		req.Name = reqIn.Name
	}

	if reqIn.NumberOfCommits > 0 && req.NumberOfCommits != reqIn.NumberOfCommits {
		req.NumberOfCommits = reqIn.NumberOfCommits
		for _, check := range req.Checks {
			check.Pass = pb.PullRequest_Check_FAIL
		}
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

	s.CtxLog(ctx, fmt.Sprintf("%v", s.processPullRequest(ctx, req)))

	return &pb.UpdateResponse{}, nil
}

// UpdatePullRequest updates the pull request
func (s *Server) UpdatePullRequest(ctx context.Context, req *pb.UpdateRequest) (*pb.UpdateResponse, error) {
	config, err := s.load(ctx)
	if err != nil {
		return nil, err
	}
	defer s.save(ctx, config)
	s.CtxLog(ctx, fmt.Sprintf("Update: %v", req))
	time.Sleep(time.Second * 2)
	if len(req.Update.Url) > 0 {
		for _, pr := range config.Tracking {
			if pr.Url == req.Update.Url {
				return s.update(ctx, pr, req.Update)
			}
		}
		config.Tracking = append(config.Tracking, req.Update)
		return &pb.UpdateResponse{}, nil
	}

	if len(req.Update.Shas) > 0 {
		for _, pr := range config.Tracking {
			for _, sha := range pr.Shas {
				if sha == req.Update.Shas[0] {
					return s.update(ctx, pr, req.Update)
				}
			}
		}
	}

	return nil, fmt.Errorf("Unable to locate PR %v", req)
}
