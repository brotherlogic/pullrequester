package main

import (
	"context"
	"testing"

	pbgh "github.com/brotherlogic/githubcard/proto"
	"github.com/brotherlogic/keystore/client"
	"google.golang.org/grpc"

	pb "github.com/brotherlogic/pullrequester/proto"
)

type testGithub struct {
	dial func(server string) (*grpc.ClientConn, error)
}

func (p *testGithub) getPullRequest(ctx context.Context, req *pbgh.PullRequest) (*pbgh.PullResponse, error) {
	return &pbgh.PullResponse{}, nil
}

func (p *testGithub) closePullRequest(ctx context.Context, req *pbgh.CloseRequest) (*pbgh.CloseResponse, error) {
	return &pbgh.CloseResponse{}, nil
}

func InitTest() *Server {
	s := Init()
	s.SkipLog = true
	s.GoServer.KSclient = *keystoreclient.GetTestClient("./testing")
	s.github = &testGithub{}

	return s
}

func TestAddPlainUpdate(t *testing.T) {
	s := InitTest()

	_, err := s.UpdatePullRequest(context.Background(), &pb.UpdateRequest{Update: &pb.PullRequest{NumberOfCommits: 5, Shas: []string{"sha2"}, Checks: []*pb.PullRequest_Check{&pb.PullRequest_Check{Source: "blahs"}}}})
	if err == nil {
		t.Errorf("Missing commit did not fail")
	}

}
func TestAddUpdate(t *testing.T) {
	s := InitTest()

	_, err := s.UpdatePullRequest(context.Background(), &pb.UpdateRequest{Update: &pb.PullRequest{Url: "https://api.github.com/repos/brotherlogic/pullrequester/pulls/11", Shas: []string{"sha1"}}})
	if err != nil {
		t.Errorf("Bad update: %v", err)
	}

	if len(s.config.Tracking) != 1 {
		t.Errorf("New Pull request was not tracked!")
	}

	_, err = s.UpdatePullRequest(context.Background(), &pb.UpdateRequest{Update: &pb.PullRequest{Url: "https://api.github.com/repos/brotherlogic/pullrequester/pulls/11", Shas: []string{"sha2"}}})

	if err != nil {
		t.Errorf("New Pull request was not tracked!")
	}

	if len(s.config.Tracking) != 1 {
		t.Errorf("New pull request was added not appended")
	}

	_, err = s.UpdatePullRequest(context.Background(), &pb.UpdateRequest{Update: &pb.PullRequest{Url: "https://api.github.com/repos/brotherlogic/pullrequester/pulls/11", NumberOfCommits: 5, Checks: []*pb.PullRequest_Check{&pb.PullRequest_Check{Source: "blahs", Pass: pb.PullRequest_Check_PASS}}}})

	if err != nil {
		t.Errorf("New Pull request was not tracked!")
	}

	if len(s.config.Tracking) != 1 {
		t.Errorf("New pull request was added not appended")
	}

	_, err = s.UpdatePullRequest(context.Background(), &pb.UpdateRequest{Update: &pb.PullRequest{Url: "https://api.github.com/repos/brotherlogic/pullrequester/pulls/11", NumberOfCommits: 5, Checks: []*pb.PullRequest_Check{&pb.PullRequest_Check{Source: "blahs", Pass: pb.PullRequest_Check_PASS}}}})

	if len(s.config.Tracking[0].Checks) != 1 {
		t.Errorf("Too many checks!")
	}

	_, err = s.UpdatePullRequest(context.Background(), &pb.UpdateRequest{Update: &pb.PullRequest{NumberOfCommits: 5, Shas: []string{"sha2"}, Checks: []*pb.PullRequest_Check{&pb.PullRequest_Check{Source: "blahtoo", Pass: pb.PullRequest_Check_FAIL}}}})

	if err != nil {
		t.Errorf("New Pull request was not tracked!")
	}

	if len(s.config.Tracking) != 1 {
		t.Errorf("Append happened this time")
	}

	if len(s.config.Tracking[0].Checks) != 2 {
		t.Errorf("Too many checks!")
	}

	if s.config.Tracking[0].Checks[0].Pass == pb.PullRequest_Check_PASS {
		t.Errorf("Prior pass should be nulled out")
	}

}

func TestProcessPullRequest(t *testing.T) {
	s := InitTest()

	err := s.processPullRequest(context.Background(), &pb.PullRequest{NumberOfCommits: 1, Checks: []*pb.PullRequest_Check{&pb.PullRequest_Check{Pass: pb.PullRequest_Check_FAIL}, &pb.PullRequest_Check{Pass: pb.PullRequest_Check_PASS}}})
	if err == nil {
		t.Fatalf("Should have failed")
	}

	err = s.processPullRequest(context.Background(), &pb.PullRequest{Shas: []string{"sha1"}, Url: "https://api.github.com/repos/brotherlogic/pullrequester/pulls/11", NumberOfCommits: 1, Checks: []*pb.PullRequest_Check{&pb.PullRequest_Check{Pass: pb.PullRequest_Check_PASS}, &pb.PullRequest_Check{Pass: pb.PullRequest_Check_PASS}}})
	if err != nil {
		t.Errorf("Should have passed")
	}

}
