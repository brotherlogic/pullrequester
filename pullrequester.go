package main

import (
	"strconv"
	"strings"

	"github.com/brotherlogic/goserver"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pbgh "github.com/brotherlogic/githubcard/proto"
	pbg "github.com/brotherlogic/goserver/proto"
	pb "github.com/brotherlogic/pullrequester/proto"
)

type github interface {
	getPullRequest(ctx context.Context, req *pbgh.PullRequest) (*pbgh.PullResponse, error)
	closePullRequest(ctx context.Context, req *pbgh.CloseRequest) (*pbgh.CloseResponse, error)
}

type prodGithub struct {
	dial func(ctx context.Context, server string) (*grpc.ClientConn, error)
}

func (p *prodGithub) getPullRequest(ctx context.Context, req *pbgh.PullRequest) (*pbgh.PullResponse, error) {
	conn, err := p.dial(ctx, "githubcard")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := pbgh.NewGithubClient(conn)
	return client.GetPullRequest(ctx, req)
}

func (p *prodGithub) closePullRequest(ctx context.Context, req *pbgh.CloseRequest) (*pbgh.CloseResponse, error) {
	conn, err := p.dial(ctx, "githubcard")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := pbgh.NewGithubClient(conn)
	return client.ClosePullRequest(ctx, req)
}

const (
	// KEY - where we store sale info
	KEY = "/github.com/brotherlogic/pullrequester/config"
)

//Server main server type
type Server struct {
	*goserver.GoServer
	github github
}

func (s *Server) cleanTracking(ctx context.Context, config *pb.Config) error {
	for i, pr := range config.Tracking {
		elems := strings.Split(pr.Url, "/")
		if len(elems) > 7 {
			val, _ := strconv.Atoi(elems[7])
			prs, err := s.github.getPullRequest(ctx, &pbgh.PullRequest{Job: elems[5], PullNumber: int32(val)})

			if err != nil {
				return err
			}

			if !prs.IsOpen {
				config.Tracking = append(config.Tracking[:i], config.Tracking[i+1:]...)
				return s.cleanTracking(ctx, config)
			}
		} else {
			config.Tracking = append(config.Tracking[:i], config.Tracking[i+1:]...)
			return s.cleanTracking(ctx, config)
		}
	}

	return nil
}

// Init builds the server
func Init() *Server {
	s := &Server{
		GoServer: &goserver.GoServer{},
	}
	s.github = &prodGithub{dial: s.FDialServer}
	return s
}

func (s *Server) save(ctx context.Context, config *pb.Config) {
	s.KSclient.Save(ctx, KEY, config)
}

func (s *Server) load(ctx context.Context) (*pb.Config, error) {
	config := &pb.Config{}
	data, _, err := s.KSclient.Read(ctx, KEY, config)

	if err != nil {
		return nil, err
	}

	return data.(*pb.Config), nil
}

// DoRegister does RPC registration
func (s *Server) DoRegister(server *grpc.Server) {
	pb.RegisterPullRequesterServiceServer(server, s)
}

// ReportHealth alerts if we're not healthy
func (s *Server) ReportHealth() bool {
	return true
}

// Shutdown the server
func (s *Server) Shutdown(ctx context.Context) error {
	return nil
}

// Mote promotes/demotes this server
func (s *Server) Mote(ctx context.Context, master bool) error {
	return nil
}

// GetState gets the state of the server
func (s *Server) GetState() []*pbg.State {
	return []*pbg.State{}
}

func main() {
	server := Init()
	server.PrepServer("pullrequester")
	server.Register = server
	err := server.RegisterServerV2(false)
	if err != nil {
		return
	}

	server.Serve()
}
