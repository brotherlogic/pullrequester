package main

import (
	"context"
	"testing"

	"github.com/brotherlogic/keystore/client"

	pb "github.com/brotherlogic/pullrequester/proto"
)

func InitTest() *Server {
	s := Init()
	s.SkipLog = true
	s.GoServer.KSclient = *keystoreclient.GetTestClient("./testing")

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

	_, err := s.UpdatePullRequest(context.Background(), &pb.UpdateRequest{Update: &pb.PullRequest{Url: "blah", Shas: []string{"sha1"}}})
	if err != nil {
		t.Errorf("Bad update: %v", err)
	}

	if len(s.config.Tracking) != 1 {
		t.Errorf("New Pull request was not tracked!")
	}

	_, err = s.UpdatePullRequest(context.Background(), &pb.UpdateRequest{Update: &pb.PullRequest{Url: "blah", Shas: []string{"sha2"}}})

	if err != nil {
		t.Errorf("New Pull request was not tracked!")
	}

	if len(s.config.Tracking) != 1 {
		t.Errorf("New pull request was added not appended")
	}

	_, err = s.UpdatePullRequest(context.Background(), &pb.UpdateRequest{Update: &pb.PullRequest{Url: "blah", NumberOfCommits: 5, Checks: []*pb.PullRequest_Check{&pb.PullRequest_Check{Source: "blahs", Pass: pb.PullRequest_Check_PASS}}}})

	if err != nil {
		t.Errorf("New Pull request was not tracked!")
	}

	if len(s.config.Tracking) != 1 {
		t.Errorf("New pull request was added not appended")
	}

	_, err = s.UpdatePullRequest(context.Background(), &pb.UpdateRequest{Update: &pb.PullRequest{Url: "blah", NumberOfCommits: 5, Checks: []*pb.PullRequest_Check{&pb.PullRequest_Check{Source: "blahs", Pass: pb.PullRequest_Check_PASS}}}})

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
